package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
)

type SpotifyAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type SpotifyTrack struct {
	Title       string
	AlbumCover  string
	AlbumName   string
	ArtistName  string
	ReleaseDate string
	SpotifyLink string
}

type SpotifyAlbum struct {
	Name          string
	CoverImage    string
	ReleaseDate   string
	NumberOfSongs int
	SpotifyLink   string
}

func getAccessToken() (string, error) {
	clientID := "87b5f30848de48eab5d2b183fcfc0591"
	clientSecret := "8618874ace51434791d243d72c322cf2"
	baseURL := "https://accounts.spotify.com/api/token"

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", baseURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)
	req.URL.RawQuery = data.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp SpotifyAccessTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func getTrackInfo(accessToken string) (*SpotifyTrack, error) {
	baseURL := "https://api.spotify.com/v1/tracks/0EzNyXyU7gHzj2TN8qYThj"

	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get track info. Status code: %d", resp.StatusCode)
	}
	var trackData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&trackData); err != nil {
		return nil, err
	}

	trackInfo := &SpotifyTrack{}
	trackInfo.Title = trackData["name"].(string)
	trackInfo.AlbumCover = trackData["album"].(map[string]interface{})["images"].([]interface{})[0].(map[string]interface{})["url"].(string)
	trackInfo.AlbumName = trackData["album"].(map[string]interface{})["name"].(string)
	artists := trackData["artists"].([]interface{})
	if len(artists) > 0 {
		trackInfo.ArtistName = artists[0].(map[string]interface{})["name"].(string)
	}
	trackInfo.ReleaseDate = trackData["album"].(map[string]interface{})["release_date"].(string)
	trackInfo.SpotifyLink = trackData["external_urls"].(map[string]interface{})["spotify"].(string)

	return trackInfo, nil
}

func getAlbumInfo(accessToken string) ([]SpotifyAlbum, error) {
	baseURL := "https://api.spotify.com/v1/artists/3IW7ScrzXmPvZhB27hmfgy/albums"

	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get album info. Status code: %d", resp.StatusCode)
	}

	var albumsData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&albumsData); err != nil {
		return nil, err
	}
	albumsList := albumsData["items"].([]interface{})
	var albums []SpotifyAlbum // Slice d'objets SpotifyAlbum

	for _, item := range albumsList {
		album := item.(map[string]interface{})
		var newAlbum SpotifyAlbum
		newAlbum.Name = album["name"].(string)
		images := album["images"].([]interface{})
		if len(images) > 0 {
			newAlbum.CoverImage = images[0].(map[string]interface{})["url"].(string)
		}
		newAlbum.ReleaseDate = album["release_date"].(string)
		tracks := album["total_tracks"].(float64)
		newAlbum.NumberOfSongs = int(tracks)
		newAlbum.SpotifyLink = album["external_urls"].(map[string]interface{})["spotify"].(string)

		albums = append(albums, newAlbum) // Ajout de l'album à la liste
	}
	return albums, nil
}

func sdmHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken()
	if err != nil {
		http.Error(w, "Erreur lors de la récupération du token d'accès", http.StatusInternalServerError)
		return
	}

	trackInfo, err := getTrackInfo(accessToken)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des informations de la piste", http.StatusInternalServerError)
		return
	}

	data := SpotifyTrack{
		Title:       trackInfo.Title,
		AlbumCover:  trackInfo.AlbumCover,
		AlbumName:   trackInfo.AlbumName,
		ArtistName:  trackInfo.ArtistName,
		ReleaseDate: trackInfo.ReleaseDate,
		SpotifyLink: trackInfo.SpotifyLink,
	}
	renderTemplate(w, "sdm", data)
}

func julHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := getAccessToken()
	if err != nil {
		http.Error(w, "Erreur lors de la récupération du token d'accès", http.StatusInternalServerError)
		return
	}

	albumsInfo, err := getAlbumInfo(accessToken)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des informations des albums", http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "jul", albumsInfo)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/track/sdm", sdmHandler)
	http.HandleFunc("/album/jul", julHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	// Taken from hangman
	tmpl, err := template.New(tmplName).Funcs(template.FuncMap{"join": join}).ParseFiles(tmplName + ".html")
	if err != nil {
		fmt.Println("Error parsing template:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, tmplName, data)
	if err != nil {
		fmt.Println("Error executing template:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
func join(s []string, sep string) string {
	// same
	return strings.Join(s, sep)
}
