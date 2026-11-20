// Adapterهای بین‌ماژولی — تنها نقطه اتصال قراردادهای داخلی ماژول‌ها.
package bootstrap

import (
	"context"
	"errors"

	assetsApp "fanavaryar/internal/modules/assets/application"
	locationsApp "fanavaryar/internal/modules/locations/application"
	locationsDomain "fanavaryar/internal/modules/locations/domain"
	usersApp "fanavaryar/internal/modules/users/application"
	"fanavaryar/internal/platform/auth"
)

// locationReader قرارداد assets.LocationReader را با سرویس ماژول Locations
// برآورده می‌کند و خطای «یافت نشد» را به خطای قراردادی ماژول تجهیزات ترجمه می‌کند.
type locationReader struct {
	svc *locationsApp.Service
}

func (l locationReader) GetSnapshot(ctx context.Context, id int64) (assetsApp.LocationSnapshot, error) {
	kind, name, err := l.svc.Snapshot(ctx, id)
	if err != nil {
		if errors.Is(err, locationsDomain.ErrNotFound) {
			return assetsApp.LocationSnapshot{}, assetsApp.ErrLocationNotFound
		}
		return assetsApp.LocationSnapshot{}, err
	}
	return assetsApp.LocationSnapshot{ID: id, Kind: kind, Name: name}, nil
}

// scopeLocationChecker قرارداد بررسی مکانِ محدوده دسترسی کاربران را به
// ماژول Locations وصل می‌کند.
type scopeLocationChecker struct {
	svc *locationsApp.Service
}

func (c scopeLocationChecker) ScopeLocationKind(ctx context.Context, id int64) (string, error) {
	return c.svc.ScopeLocationKind(ctx, id)
}

// passwordHasher قرارداد usersApp.PasswordHasher را به پکیج platform/auth وصل می‌کند.
type passwordHasher struct{}

func (passwordHasher) Hash(password string) (string, error) {
	return auth.HashPassword(password)
}

func (passwordHasher) Verify(password, encoded string) (bool, error) {
	return auth.VerifyPassword(password, encoded)
}

// نگه‌داشتن قرارداد سرویس کاربران برای تزریق چک‌کننده مکان در bootstrap
var _ usersApp.ScopeLocationChecker = scopeLocationChecker{}
