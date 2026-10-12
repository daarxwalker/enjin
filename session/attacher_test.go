package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/security"
	"github.com/daarxwalker/enjin/token"

	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	entries map[string][]byte
}

func newFakeCache() *fakeCache {
	return &fakeCache{entries: make(map[string][]byte)}
}

func (c *fakeCache) GetJSON(_ context.Context, key string, t any) (bool, error) {
	data, ok := c.entries[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(data, t)
}

func (c *fakeCache) SetJSON(_ context.Context, key string, v any, _ time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.entries[key] = data
	return nil
}

func (c *fakeCache) UpdateExpiration(_ context.Context, key string, _ time.Duration) (bool, error) {
	_, ok := c.entries[key]
	return ok, nil
}

func (c *fakeCache) Delete(_ context.Context, key string) error {
	delete(c.entries, key)
	return nil
}

func TestAttach(t *testing.T) {
	admin := &security.Role{Name: "admin"}
	editor := &security.Role{Name: "editor"}
	roles := map[string]*security.Role{admin.Name: admin, editor.Name: editor}

	setup := func(t *testing.T, current *Profile) (*Store, *fakeCache) {
		t.Helper()
		cache := newFakeCache()
		store := NewStore(cache, WithRoles(roles))
		if current != nil {
			require.NoError(t, saveProfileToCache(context.Background(), cache, current, store.cookie.TTL))
		}
		return store, cache
	}

	attach := func(t *testing.T, store *Store, token string, options ...AttachOption) (*Profile, string) {
		t.Helper()
		var attached *Profile
		handler := store.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attached = Attach(w, r, options...)
		}))
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if token != "" {
			r.AddCookie(&http.Cookie{Name: store.cookie.Name, Value: token})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var cookie string
		for _, c := range w.Result().Cookies() {
			if c.Name == store.cookie.Name {
				cookie = c.Value
			}
		}
		return attached, cookie
	}

	cached := func(t *testing.T, cache *fakeCache, token string) *Profile {
		t.Helper()
		profile, err := getProfileFromCache(context.Background(), cache, token, roles)
		require.NoError(t, err)
		return profile
	}

	t.Run(
		"create session", func(t *testing.T) {
			store, cache := setup(t, nil)
			profile, cookie := attach(t, store, "", WithID("user"))
			require.NotEmpty(t, profile.Token)
			require.Equal(t, profile.Token, cookie)
			require.Equal(t, "user", cached(t, cache, profile.Token).ID)
		},
	)

	t.Run(
		"rotate token for same user", func(t *testing.T) {
			old := token.New()
			current := &Profile{ID: "user", Role: admin, Token: old, Values: map[string]string{"theme": "dark"}}
			store, cache := setup(t, current)
			profile, cookie := attach(t, store, old, WithID("user"), WithRole(editor))
			require.NotEqual(t, old, profile.Token)
			require.Equal(t, profile.Token, cookie)
			require.Nil(t, cached(t, cache, old))
			stored := cached(t, cache, profile.Token)
			require.Equal(t, editor, stored.Role)
			require.Equal(t, map[string]string{"theme": "dark"}, stored.Values)
		},
	)

	t.Run(
		"rotate token without identity option", func(t *testing.T) {
			old := token.New()
			current := &Profile{ID: "user", Key: "old@example.com", Role: admin, Token: old}
			store, cache := setup(t, current)
			profile, cookie := attach(t, store, old, WithKey("new@example.com"))
			require.NotEqual(t, old, profile.Token)
			require.Equal(t, profile.Token, cookie)
			require.Nil(t, cached(t, cache, old))
			stored := cached(t, cache, profile.Token)
			require.Equal(t, "user", stored.ID)
			require.Equal(t, "new@example.com", stored.Key)
			require.Equal(t, admin, stored.Role)
		},
	)

	t.Run(
		"start clean session for other user", func(t *testing.T) {
			old := token.New()
			current := &Profile{ID: "admin", Role: admin, Token: old, Values: map[string]string{"theme": "dark"}}
			store, cache := setup(t, current)
			profile, cookie := attach(t, store, old, WithID("user"), WithKey("user@example.com"))
			require.NotEqual(t, old, profile.Token)
			require.Equal(t, profile.Token, cookie)
			require.Nil(t, cached(t, cache, old))
			stored := cached(t, cache, profile.Token)
			require.Equal(t, "user", stored.ID)
			require.Nil(t, stored.Role)
			require.Empty(t, stored.Values)
		},
	)
}

func TestAttachBearer(t *testing.T) {
	admin := &security.Role{Name: "admin"}
	roles := map[string]*security.Role{admin.Name: admin}

	setup := func(t *testing.T, current *Profile) (*Store, *fakeCache) {
		t.Helper()
		cache := newFakeCache()
		store := NewStore(cache, WithRoles(roles), WithBearer())
		if current != nil {
			require.NoError(t, saveProfileToCache(context.Background(), cache, current, store.cookie.TTL))
		}
		return store, cache
	}

	serve := func(t *testing.T, store *Store, token string, handle func(w http.ResponseWriter, r *http.Request)) []*http.Cookie {
		t.Helper()
		handler := store.Middleware(http.HandlerFunc(handle))
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Result().Cookies()
	}

	t.Run(
		"keep token", func(t *testing.T) {
			current := &Profile{ID: "user", Token: token.New()}
			store, cache := setup(t, current)
			var attached *Profile
			cookies := serve(t, store, current.Token, func(w http.ResponseWriter, r *http.Request) {
				attached = Attach(w, r, WithRole(admin))
			})
			require.Empty(t, cookies)
			require.Equal(t, current.Token, attached.Token)
			profile, err := getProfileFromCache(context.Background(), cache, current.Token, roles)
			require.NoError(t, err)
			require.Equal(t, admin, profile.Role)
		},
	)

	t.Run(
		"destroy without cookie", func(t *testing.T) {
			current := &Profile{ID: "user", Token: token.New()}
			store, cache := setup(t, current)
			cookies := serve(t, store, current.Token, Destroy)
			require.Empty(t, cookies)
			require.Empty(t, cache.entries)
		},
	)
}
