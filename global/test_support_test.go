package global

import (
	"github.com/Zany2/dtoken-go"
	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

var NewBuilder = dtoken.NewBuilder

func buildTestManager(b *dtoken.Builder) (*manager.Manager, error) {
	auth, err := b.Build()
	if err != nil {
		return nil, err
	}
	return auth.Manager(), nil
}
func mustBuildTestManager(b *dtoken.Builder) *manager.Manager {
	mgr, err := buildTestManager(b)
	if err != nil {
		panic(err)
	}
	return mgr
}

func requireTestManager(auth *Auth) (*manager.Manager, error) {
	mgr := auth.Manager()
	if mgr == nil || mgr.IsClosed() {
		return nil, derror.ErrManagerNotFound
	}
	return mgr, nil
}
