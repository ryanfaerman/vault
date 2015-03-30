package vault

import (
	"errors"
	"sync"
)

var (
	NotFoundError  = errors.New("key not found")
	KeyExistsError = errors.New("key exists")
)

// Keyer represents anything that can be stored in the Vault
type Keyer interface {
	// Key should always return the same key for the same object
	// If it doesn't, you'll get multiple copies
	Key() string
}

// Persister represents long term storage
type Persister interface {
	// Persist may be called at any time. It may not receive
	// the entire vault.
	Persist(map[string]Keyer) error

	// Load should retreive the entire dataset from the persister.
	// Generally, this will only be called when the application starts.
	Load() (map[string]Keyer, error)
}

type FilterFunc func(Keyer) bool

// PersistanceError captures all the errors that occur during a
// persistance attempt. Any of the persisters may have errors.
// Receiving a PersistanceError may not mean that persistance has
// failed entirely, just that one or more Persisters have failed.
type PersistanceError struct {
	Errors []error
}

func (p *PersistanceError) Error() string {
	err_msg := "One or more persistance errors occured: \n"

	for _, err := range p.Errors {
		err_msg = err_msg + "\t" + err.Error() + "\n"
	}

	return err_msg
}

type Vault struct {
	mutex sync.Mutex
	vault map[string]Keyer

	persisters []Persister
}

func New() *Vault {
	return &Vault{
		vault:      make(map[string]Keyer),
		persisters: make([]Persister, 0),
	}
}

// Register lets you add a Persister
func (v *Vault) Register(s Persister) {
	v.persisters = append(v.persisters, s)
}

// Persist gives a copy of the vault to each persister.
// The actual persistance occurs in parallel and this method
// will not return until all have finished their tasks.
func (v *Vault) Persist() error {
	errs := make(chan error)
	vault := v.vault

	for _, store := range v.persisters {
		go func() {
			errs <- store.Persist(vault)
		}()
	}

	var p_errs *PersistanceError
	for range v.persisters {
		select {
		case err := <-errs:
			if err != nil {
				if p_errs == nil {
					p_errs = &PersistanceError{}
				}
				p_errs.Errors = append(p_errs.Errors, err)
			}
		}
	}

	if p_errs != nil {
		return p_errs
	}

	return nil
}

// Load retrieves a persisted vault from every persister.
// This happens serially and generally only occurs when the
// application starts.
func (v *Vault) Load() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for _, store := range v.persisters {
		values, err := store.Load()
		if err != nil {
			return err
		}

		for key, value := range values {
			v.vault[key] = value
		}
	}

	return nil
}

// Exists is a simple presence check for a key in the vault
func (v *Vault) Exists(key string) bool {
	_, ok := v.vault[key]
	return ok
}

// Put performs a thread-safe write of one or more
// Keyers to the vault.
func (v *Vault) Put(items ...Keyer) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for _, item := range items {
		v.vault[item.Key()] = item
	}

	return nil
}

// Get retrieves the value matching the given key.
func (v *Vault) Get(key string) (Keyer, error) {
	if !v.Exists(key) {
		return nil, NotFoundError
	}

	return v.vault[key], nil
}

// Filter the vault with the given FilterFunc
func (v *Vault) Filter(f FilterFunc) map[string]Keyer {
	filtered := map[string]Keyer{}

	for key, value := range v.vault {
		if f(value) {
			filtered[key] = value
		}
	}

	return filtered
}
