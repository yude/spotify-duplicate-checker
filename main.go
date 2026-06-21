package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	spotifyAccounts = "https://accounts.spotify.com/api/token"
	spotifyAPI      = "https://api.spotify.com/v1"
)

type authResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type playlistTracksResponse struct {
	Items  []playlistTrack `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Next   *string         `json:"next"`
}

type playlistTrack struct {
	Track track `json:"track"`
}

type track struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Album   album    `json:"album"`
	Artists []artist `json:"artists"`
}

type album struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type artist struct {
	Name string `json:"name"`
}

type duplicateGroup struct {
	TrackName string
	Entries   []entry
}

type entry struct {
	TrackID    string
	AlbumName  string
	ArtistName string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: spotify-duplicate-checker <playlist-id-or-url>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "environment variables:")
		fmt.Fprintln(os.Stderr, "  SPOTIFY_CLIENT_ID     (required)")
		fmt.Fprintln(os.Stderr, "  SPOTIFY_CLIENT_SECRET (required)")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "get credentials at https://developer.spotify.com/dashboard")
		os.Exit(1)
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		fmt.Fprintln(os.Stderr, "SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
		os.Exit(1)
	}

	playlistID := extractPlaylistID(os.Args[1])

	token, err := getAccessToken(clientID, clientSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auth failed: %v\n", err)
		os.Exit(1)
	}

	tracks, err := fetchAllTracks(playlistID, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("fetched %d tracks\n", len(tracks))

	duplicates := findDuplicates(tracks)

	if len(duplicates) == 0 {
		fmt.Println("no duplicate track names found across different albums")
		return
	}

	fmt.Printf("\nfound %d track name(s) appearing on different albums:\n\n", len(duplicates))
	for _, d := range duplicates {
		fmt.Printf("  %q\n", d.TrackName)
		for _, e := range d.Entries {
			fmt.Printf("    - album: %q  artist: %s  id: %s\n", e.AlbumName, e.ArtistName, e.TrackID)
		}
		fmt.Println()
	}
}

func extractPlaylistID(input string) string {
	if !strings.Contains(input, "/") {
		return input
	}

	u, err := url.Parse(input)
	if err != nil {
		return input
	}

	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	for i, p := range parts {
		if p == "playlist" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return input
}

func getAccessToken(clientID, clientSecret string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", spotifyAccounts, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, body)
	}

	var ar authResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return "", err
	}

	return ar.AccessToken, nil
}

func fetchAllTracks(playlistID, token string) ([]track, error) {
	var all []track
	offset := 0
	limit := 100

	for {
		u := fmt.Sprintf("%s/playlists/%s/tracks?offset=%d&limit=%d&fields=next,total,items(track(id,name,album(id,name),artists(name)))",
			spotifyAPI, playlistID, offset, limit)

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("playlist request returned %d: %s", resp.StatusCode, body)
		}

		var ptr playlistTracksResponse
		if err := json.NewDecoder(resp.Body).Decode(&ptr); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for _, item := range ptr.Items {
			if item.Track.ID != "" {
				all = append(all, item.Track)
			}
		}

		if ptr.Next == nil {
			break
		}
		offset += limit

		time.Sleep(50 * time.Millisecond)
	}

	return all, nil
}

func findDuplicates(tracks []track) []duplicateGroup {
	type nameEntry struct {
		name    string
		entries []entry
	}

	byName := map[string]*nameEntry{}

	for _, t := range tracks {
		artistName := ""
		if len(t.Artists) > 0 {
			artistName = t.Artists[0].Name
		}
		e := entry{
			TrackID:    t.ID,
			AlbumName:  t.Album.Name,
			ArtistName: artistName,
		}
		key := strings.ToLower(t.Name)

		if ne, ok := byName[key]; ok {
			ne.entries = append(ne.entries, e)
		} else {
			byName[key] = &nameEntry{name: t.Name, entries: []entry{e}}
		}
	}

	var result []duplicateGroup
	for _, ne := range byName {
		if len(ne.entries) < 2 {
			continue
		}

		albums := map[string]bool{}
		for _, e := range ne.entries {
			albums[e.AlbumName] = true
		}
		if len(albums) < 2 {
			continue
		}

		result = append(result, duplicateGroup{
			TrackName: ne.name,
			Entries:   ne.entries,
		})
	}

	return result
}
