package session_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mathias/boeuf/internal/session"
)

func TestNewStore(t *testing.T) {
	// ARRANGE
	sessionKey := "test-session-key-32-bytes-long!!"

	// ACT
	store := session.NewStore(sessionKey, false)

	// ASSERT
	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}

func TestSessionStorage(t *testing.T) {
	// ARRANGE
	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, false)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// ACT - Save data to session
	sess, err := store.Get(req, "boeuf-session")
	if err != nil {
		t.Fatalf("Get session failed: %v", err)
	}

	sess.Values["test_key"] = "test_value"
	sess.Values["code_verifier"] = "abcd1234"

	err = sess.Save(req, w)
	if err != nil {
		t.Fatalf("Save session failed: %v", err)
	}

	// ASSERT - Verify cookie was set
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("Expected session cookie to be set")
	}

	foundSessionCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "boeuf-session" {
			foundSessionCookie = true
			if !cookie.HttpOnly {
				t.Error("Expected HttpOnly flag to be true")
			}
			// In test mode (secure=false), Secure flag should be false
			if cookie.Secure {
				t.Error("Expected Secure flag to be false in test mode")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Errorf("Expected SameSite=Lax, got %v", cookie.SameSite)
			}
		}
	}

	if !foundSessionCookie {
		t.Error("Expected boeuf-session cookie")
	}
}

func TestSecureCookieInProduction(t *testing.T) {
	// ARRANGE
	sessionKey := "test-session-key-32-bytes-long!!"
	store := session.NewStore(sessionKey, true) // secure = true for production

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// ACT
	sess, _ := store.Get(req, "boeuf-session")
	sess.Values["data"] = "value"
	sess.Save(req, w)

	// ASSERT
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "boeuf-session" {
			if !cookie.Secure {
				t.Error("Expected Secure flag to be true in production mode")
			}
		}
	}
}
