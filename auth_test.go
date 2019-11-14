package potoq

import "testing"
import "github.com/google/uuid"

func TestOfflinePlayerUUID(t *testing.T) {
	var table = map[string]uuid.UUID{
		"Player1234": uuid.MustParse("00e45140-b547-315c-90a2-74e494a239e1"),
		"CubixPirat": uuid.MustParse("cf63089e-8b93-345a-b3e3-80416f16af13"),
		"cubixpirat": uuid.MustParse("2b792902-f4be-3596-914f-9d56b9f22565"),
	}
	for nickname, expected := range table {
		v := OfflinePlayerUUID(nickname)
		if v != expected {
			t.Errorf("%s: got %s expected %s", nickname, v, expected)
		}
	}
}
