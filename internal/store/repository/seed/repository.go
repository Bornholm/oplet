package seed

import "github.com/bornholm/oplet/internal/store"

type Repository struct {
	store *store.Store
}

func NewRepository(store *store.Store) *Repository {
	return &Repository{
		store: store,
	}
}
