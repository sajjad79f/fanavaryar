package application_test

import (
        "context"
        "errors"
        "testing"
        "time"

        "fanavaryar/internal/modules/locations/application"
        "fanavaryar/internal/modules/locations/domain"
        "fanavaryar/internal/platform/auth"
)

// fakeRepo پیاده‌سازی درون‌حافظه‌ای Repository برای تست منطق سرویس بدون دیتابیس.
type fakeRepo struct {
        seq      int64
        kinds    map[int64]string
        names    map[int64]string
        parents  map[int64]*int64
        children map[int64]int
        assets   map[int64]int
        deleted  []int64
}

func newFakeRepo() *fakeRepo {
        return &fakeRepo{
                kinds:    map[int64]string{},
                names:    map[int64]string{},
                parents:  map[int64]*int64{},
                children: map[int64]int{},
                assets:   map[int64]int{},
        }
}

func (r *fakeRepo) List(ctx context.Context) ([]application.LocationRow, error) {
        return nil, nil
}

func (r *fakeRepo) GetKind(ctx context.Context, id int64) (string, error) {
        k, ok := r.kinds[id]
        if !ok {
                return "", domain.ErrNotFound
        }
        return k, nil
}

func (r *fakeRepo) GetSnapshot(ctx context.Context, id int64) (string, string, error) {
        k, err := r.GetKind(ctx, id)
        if err != nil {
                return "", "", err
        }
        return k, r.names[id], nil
}

func (r *fakeRepo) Create(ctx context.Context, in application.CreateInput) (application.LocationRow, error) {
        r.seq++
        r.kinds[r.seq] = in.Kind
        r.names[r.seq] = in.Name
        r.parents[r.seq] = in.ParentID
        if in.ParentID != nil {
                r.children[*in.ParentID]++
        }
        return application.LocationRow{ID: r.seq, Kind: in.Kind, Name: in.Name, CreatedAt: time.Now()}, nil
}

func (r *fakeRepo) Rename(ctx context.Context, id int64, name string, code *string) (application.LocationRow, error) {
        if _, ok := r.kinds[id]; !ok {
                return application.LocationRow{}, domain.ErrNotFound
        }
        r.names[id] = name
        return application.LocationRow{ID: id, Name: name, CreatedAt: time.Now()}, nil
}

func (r *fakeRepo) Delete(ctx context.Context, id int64) error {
        if _, ok := r.kinds[id]; !ok {
                return domain.ErrNotFound
        }
        delete(r.kinds, id)
        r.deleted = append(r.deleted, id)
        return nil
}

func (r *fakeRepo) CountChildren(ctx context.Context, id int64) (int, error) {
        return r.children[id], nil
}

func (r *fakeRepo) CountAssets(ctx context.Context, id int64) (int, error) {
        return r.assets[id], nil
}

func TestCreateNestingRules(t *testing.T) {
        repo := newFakeRepo()
        svc := application.NewService(repo, nil)
        ctx := context.Background()
        admin := &auth.AuthUser{ID: 1, Role: "admin"}

        // ساختمان ریشه: بدون والد مجاز
        b, err := svc.Create(ctx, admin, application.CreateInput{Name: "ساختمان مرکزی", Kind: "building"})
        if err != nil {
                t.Fatalf("root building: %v", err)
        }
        // طبقه زیر ساختمان مجاز
        if _, err = svc.Create(ctx, admin, application.CreateInput{Name: "طبقه ۱", Kind: "floor", ParentID: &b.ID}); err != nil {
                t.Fatalf("floor under building: %v", err)
        }
        // ساختمان زیر ساختمان مجاز نیست
        if _, err = svc.Create(ctx, admin, application.CreateInput{Name: "بلوک", Kind: "building", ParentID: &b.ID}); err == nil {
                t.Fatal("building under building must fail")
        }
        // رک زیر طبقه مجاز نیست
        if _, err = svc.Create(ctx, admin, application.CreateInput{Name: "رک", Kind: "rack", ParentID: &b.ID}); err == nil {
                t.Fatal("rack under building must fail")
        }
}

func TestDeleteGuards(t *testing.T) {
        repo := newFakeRepo()
        svc := application.NewService(repo, nil)
        ctx := context.Background()
        admin := &auth.AuthUser{ID: 1, Role: "admin"}

        b, _ := svc.Create(ctx, admin, application.CreateInput{Name: "ساختمان", Kind: "building"})
        fl, _ := svc.Create(ctx, admin, application.CreateInput{Name: "طبقه ۱", Kind: "floor", ParentID: &b.ID})
        room, _ := svc.Create(ctx, admin, application.CreateInput{Name: "اتاق", Kind: "room", ParentID: &fl.ID})

        // مکان دارای زیرمجموعه حذف نمی‌شود
        repo.children[b.ID] = 1
        if err := svc.Delete(ctx, admin, b.ID); err == nil {
                t.Fatal("delete with children must fail")
        }
        repo.children[b.ID] = 0

        // مکان دارای تجهیز حذف نمی‌شود
        repo.assets[room.ID] = 2
        if err := svc.Delete(ctx, admin, room.ID); err == nil {
                t.Fatal("delete with assets must fail")
        }
        repo.assets[room.ID] = 0

        if err := svc.Delete(ctx, admin, room.ID); err != nil {
                t.Fatalf("clean delete: %v", err)
        }
}

func TestSnapshotAndScopeKind(t *testing.T) {
        repo := newFakeRepo()
        svc := application.NewService(repo, nil)
        ctx := context.Background()
        admin := &auth.AuthUser{ID: 1, Role: "admin"}

        b, _ := svc.Create(ctx, admin, application.CreateInput{Name: "ساختمان", Kind: "building"})
        f, err := svc.Create(ctx, admin, application.CreateInput{Name: "طبقه ۲", Kind: "floor", ParentID: &b.ID})
        if err != nil {
                t.Fatalf("create floor: %v", err)
        }
        kind, name, err := svc.Snapshot(ctx, f.ID)
        if err != nil {
                t.Fatalf("snapshot: %v", err)
        }
        if kind != "floor" || name != "طبقه ۲" {
                t.Fatalf("snapshot = %q %q", kind, name)
        }
        got, err := svc.ScopeLocationKind(ctx, f.ID)
        if err != nil || got != "floor" {
                t.Fatalf("scope kind = %q err=%v", got, err)
        }
        if _, err := svc.ScopeLocationKind(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
                t.Fatalf("missing location must be ErrNotFound, got %v", err)
        }
}
