package vault_test

import (
	"testing"

	"github.com/nbio/st"
	"github.com/ryanfaerman/vault"
)

type Thing struct {
	Name string
	Rank int

	key string
}

func (t *Thing) Key() string {
	if t.key == "" {
		t.key = vault.Token()
	}

	return t.key
}

type MemoryStore struct {
	Data map[string]vault.Keyer
}

func (m *MemoryStore) Persist(d map[string]vault.Keyer) error {
	m.Data = d
	return nil
}

func (m *MemoryStore) Load() (map[string]vault.Keyer, error) {
	return m.Data, nil
}

func TestVaultCrud(t *testing.T) {
	the_vault := vault.New()

	some_thing := &Thing{}

	the_vault.Put(some_thing)

	a, err := the_vault.Get(some_thing.Key())

	st.Expect(t, err, nil)
	st.Expect(t, a, some_thing)

	_, err = the_vault.Get("not_a_key")
	st.Expect(t, err, vault.NotFoundError)
}

func TestVaultFiltering(t *testing.T) {
	the_vault := vault.New()

	the_vault.Put(
		&Thing{Name: "Peter", Rank: 10},
		&Thing{Name: "Piper", Rank: 11},
		&Thing{Name: "Pickle", Rank: 19},
		&Thing{Name: "Potato", Rank: 101},
		&Thing{Name: "Pencil", Rank: 98},
		&Thing{Name: "Porch", Rank: 645},
		&Thing{Name: "Patio", Rank: 3},
		&Thing{Name: "Keyboard", Rank: 88},
	)

	rs := the_vault.Filter(func(item vault.Keyer) bool {
		return item.(*Thing).Name == "Peter"
	})

	st.Expect(t, len(rs), 1)
	for _, item := range rs {
		st.Expect(t, item.(*Thing).Name, "Peter")
	}
}

func TestStoreRegistration(t *testing.T) {
	the_vault := vault.New()
	the_memory_store := &MemoryStore{Data: make(map[string]vault.Keyer)}
	the_vault.Register(the_memory_store)

	the_thing := &Thing{Name: "Peter", Rank: 10}
	the_vault.Put(the_thing)

	err := the_vault.Persist()
	st.Expect(t, err, nil)

	st.Expect(t, len(the_memory_store.Data), 1)
	st.Expect(t, the_memory_store.Data[the_thing.Key()], the_thing)

	another_vault := vault.New()
	another_vault.Register(the_memory_store)
	err = another_vault.Load()
	st.Expect(t, err, nil)

	item, err := another_vault.Get(the_thing.Key())

	st.Expect(t, err, nil)
	st.Expect(t, item, the_thing)

}

func TestVaultSize(t *testing.T) {
	the_vault := vault.New()

	st.Expect(t, the_vault.Size(), 0)

	the_vault.Put(&Thing{Name: "Peter Parker"})

	st.Expect(t, the_vault.Size(), 1)

	the_vault.Put(&Thing{Name: "Tony Stark"}, &Thing{Name: "Frank Castle"})

	st.Expect(t, the_vault.Size(), 3)
}

func BenchmarkVaultSerialFiltering(b *testing.B) {
	the_vault := vault.New()

	// Let's shove a ton of stuff into the vault
	for i := 0; i < 10000; i++ {
		the_vault.Put(&Thing{Name: vault.Token()})
	}

	// The one thing we're going to look for
	the_vault.Put(&Thing{Name: "New Orleans"})

	// Let's bury our needle, just for fun
	for i := 0; i < 10000; i++ {
		the_vault.Put(&Thing{Name: vault.Token()})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		the_vault.Filter(func(item vault.Keyer) bool {
			return item.(*Thing).Name == "New Orleans"
		})
	}

}
