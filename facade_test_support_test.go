package dtoken

import (
	derror "github.com/Zany2/dtoken-go/errors"
	manager "github.com/Zany2/dtoken-go/internal/engine"
)

func buildTestManager(b *Builder) (*manager.Manager, error) {
	auth, err := b.BuildInstance()
	if err != nil {
		return nil, err
	}
	return auth.Manager(), nil
}
func mustBuildTestManager(b *Builder) *manager.Manager {
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
