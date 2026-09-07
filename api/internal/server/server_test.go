package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ry023/semdiff-sample-app/api/internal/database"
)

func TestAuthAndTodoFlow(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	server := httptest.NewServer(New(database.NewStore(db), false))
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := server.Client()
	client.Jar = jar

	register := doJSON(t, client, http.MethodPost, server.URL+"/api/auth/register", map[string]string{
		"username": "alice",
		"password": "secret",
	})
	if register.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", register.StatusCode)
	}
	register.Body.Close()

	login := doJSON(t, client, http.MethodPost, server.URL+"/api/auth/login", map[string]string{
		"username": "alice",
		"password": "secret",
	})
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", login.StatusCode)
	}
	login.Body.Close()

	created := doJSON(t, client, http.MethodPost, server.URL+"/api/todos", map[string]string{"title": "try semdiff"})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create todo status = %d", created.StatusCode)
	}
	var todo struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(created.Body).Decode(&todo); err != nil {
		t.Fatal(err)
	}
	created.Body.Close()

	updated := doJSON(t, client, http.MethodPatch, server.URL+"/api/todos/"+strconv.FormatInt(todo.ID, 10), map[string]bool{"completed": true})
	if updated.StatusCode != http.StatusOK {
		t.Fatalf("update todo status = %d", updated.StatusCode)
	}
	updated.Body.Close()

	list := doJSON(t, client, http.MethodGet, server.URL+"/api/todos", nil)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list todo status = %d", list.StatusCode)
	}
	list.Body.Close()
}

func TestTodoRequiresAuthentication(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	server := httptest.NewServer(New(database.NewStore(db), false))
	defer server.Close()
	response := doJSON(t, http.DefaultClient, http.MethodGet, server.URL+"/api/todos", nil)
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any) *http.Response {
	t.Helper()
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, url, payload)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
