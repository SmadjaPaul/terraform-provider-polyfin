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
