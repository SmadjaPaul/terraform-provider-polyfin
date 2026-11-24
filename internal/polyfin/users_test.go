package polyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserLifecycleWithoutProduction(t *testing.T) {
	var created *User
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/api/session":
			fmt.Fprint(w, "{\"user\":{\"isAdministrator\":true}}")
		case r.URL.Path == "/admin/api/users" && r.Method == "GET":
			if created == nil {
				fmt.Fprint(w, "[]")
				return
			}
			json.NewEncoder(w).Encode([]User{*created})
		case r.URL.Path == "/admin/api/users" && r.Method == "POST":
			var in map[string]any
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Error(err)
			}
			if in["password"] != "synthetic-pass" || in["isAdministrator"] != false {
				t.Errorf("unexpected body")
			}
			created = &User{ID: "u1", Name: in["name"].(string)}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(created)
		case r.URL.Path == "/admin/api/users/u1" && r.Method == "DELETE":
			created = nil
			w.WriteHeader(204)
		default:
			http.Error(w, "unknown endpoint", 404)
		}
	}))
	defer srv.Close()
	c, err := New(srv.URL, cookie(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	u, err := c.CreateUser(ctx, "svc-test", "synthetic-pass", false, true)
	if err != nil || u.ID != "u1" {
		t.Fatalf("create: %v %+v", err, u)
	}
	if _, ok, err := c.FindUser(ctx, "u1"); err != nil || !ok {
		t.Fatalf("find: %v %v", err, ok)
	}
	if err := c.DeleteUser(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteUser(ctx, "u1"); err != nil {
		t.Fatalf("delete of a missing user must succeed: %v", err)
	}
	if _, ok, _ := c.FindUser(ctx, "u1"); ok {
		t.Fatal("user still listed")
	}
	if _, err := c.CreateUser(ctx, "x", "", false, false); err == nil {
		t.Fatal("empty password must be refused")
	}
}

func TestSetUserDownloadsSendsOnlyThatField(t *testing.T) {
	downloads := false
	var patched map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/api/session":
			fmt.Fprint(w, "{\"user\":{\"isAdministrator\":true}}")
		case r.URL.Path == "/admin/api/users" && r.Method == "GET":
			json.NewEncoder(w).Encode([]User{{ID: "u1", Name: "svc", IsHidden: true, Downloads: downloads}})
		case r.URL.Path == "/admin/api/users/u1" && r.Method == "PATCH":
			if err := json.NewDecoder(r.Body).Decode(&patched); err != nil {
				t.Error(err)
			}
			downloads = patched["downloads"] == true
			w.WriteHeader(200)
			fmt.Fprint(w, "{}")
		default:
			http.Error(w, "unknown endpoint", 404)
		}
	}))
	defer srv.Close()
	c, err := New(srv.URL, cookie(t))
	if err != nil {
		t.Fatal(err)
	}
	u, err := c.SetUserDownloads(context.Background(), "u1", true)
	if err != nil || !u.Downloads {
		t.Fatalf("set: %v %+v", err, u)
	}
	if len(patched) != 1 || patched["downloads"] != true {
		t.Fatalf("PATCH body must carry only downloads, got %v", patched)
	}
	if !u.IsHidden || u.Name != "svc" {
		t.Fatalf("other fields must be untouched: %+v", u)
	}
}

func TestSetUserDownloadsDetectsReadbackMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/api/session":
			fmt.Fprint(w, "{\"user\":{\"isAdministrator\":true}}")
		case r.URL.Path == "/admin/api/users" && r.Method == "GET":
			fmt.Fprint(w, "[{\"id\":\"u1\",\"name\":\"svc\",\"downloads\":false}]")
		default:
			fmt.Fprint(w, "{}")
		}
	}))
	defer srv.Close()
	c, _ := New(srv.URL, cookie(t))
	if _, err := c.SetUserDownloads(context.Background(), "u1", true); err == nil {
		t.Fatal("a refused change must be reported")
	}
}
