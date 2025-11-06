package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var MAPBOX_ACCESS_TOKEN = loadToken()

func loadToken() string {
	data, _ := os.ReadFile("config.json")

	var config map[string]string
	json.Unmarshal(data, &config)

	return config["mapbox_access_token"]
}

type LocationInfo struct {
	Latitude  float64
	Longitude float64
	Country   string
	Region    string
	City      string
	PlaceName string
}

type GeocodeResponse struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type Feature struct {
	Type      string    `json:"type"`
	PlaceName string    `json:"place_name"`
	Center    []float64 `json:"center"`
	Context   []Context `json:"context"`
}

type Context struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type DirectionsResponse struct {
	Routes []Route `json:"routes"`
}

type Route struct {
	Distance float64  `json:"distance"`
	Duration float64  `json:"duration"`
	Geometry Geometry `json:"geometry"`
}

type Geometry struct {
	Coordinates [][]float64 `json:"coordinates"`
}

func geocodeAddress(address string, accessToken string) (*LocationInfo, error) {
	baseURL := "https://api.mapbox.com/geocoding/v5/mapbox.places/"
	encodedAddress := url.QueryEscape(address)
	apiURL := fmt.Sprintf("%s%s.json?access_token=%s", baseURL, encodedAddress, accessToken)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("помилка HTTP запиту: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("помилка читання відповіді: %v", err)
	}

	var geocodeResp GeocodeResponse
	err = json.Unmarshal(body, &geocodeResp)
	if err != nil {
		return nil, fmt.Errorf("помилка парсингу JSON: %v", err)
	}

	if len(geocodeResp.Features) == 0 {
		return nil, fmt.Errorf("адресу не знайдено")
	}

	feature := geocodeResp.Features[0]

	location := &LocationInfo{
		Longitude: feature.Center[0],
		Latitude:  feature.Center[1],
		PlaceName: feature.PlaceName,
		Country:   "Невідомо",
		Region:    "Невідомо",
		City:      "Невідомо",
	}

	for _, ctx := range feature.Context {
		if strings.HasPrefix(ctx.ID, "country") {
			location.Country = ctx.Text
		} else if strings.HasPrefix(ctx.ID, "region") {
			location.Region = ctx.Text
		} else if strings.HasPrefix(ctx.ID, "place") {
			location.City = ctx.Text
		}
	}

	return location, nil
}

func getDistance(start, end *LocationInfo, accessToken string) (float64, float64, error) {
	baseURL := "https://api.mapbox.com/directions/v5/mapbox/driving/"
	coordinates := fmt.Sprintf("%.6f,%.6f;%.6f,%.6f",
		start.Longitude, start.Latitude,
		end.Longitude, end.Latitude)
	apiURL := fmt.Sprintf("%s%s?access_token=%s&geometries=geojson", baseURL, coordinates, accessToken)

	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, 0, fmt.Errorf("помилка HTTP запиту: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("помилка читання відповіді: %v", err)
	}

	var directionsResp DirectionsResponse
	err = json.Unmarshal(body, &directionsResp)
	if err != nil {
		return 0, 0, fmt.Errorf("помилка парсингу JSON: %v", err)
	}

	if len(directionsResp.Routes) == 0 {
		return 0, 0, fmt.Errorf("маршрут не знайдено")
	}

	distance := directionsResp.Routes[0].Distance
	duration := directionsResp.Routes[0].Duration

	return distance, duration, nil
}

func printLocation(name string, location *LocationInfo) {
	fmt.Printf("\n%s:\n", name)
	fmt.Printf("  Країна: %s\n", location.Country)
	fmt.Printf("  Область: %s\n", location.Region)
	fmt.Printf("  Місто: %s\n", location.City)
	fmt.Printf("  Широта: %.6f\n", location.Latitude)
	fmt.Printf("  Довгота: %.6f\n", location.Longitude)
	fmt.Printf("  Повна адреса: %s\n", location.PlaceName)
}

func printDistance(distance, duration float64) {
	fmt.Printf("\nІнформація про маршрут:\n")
	fmt.Printf("  Відстань: %.2f метрів (%.2f км)\n", distance, distance/1000)
	fmt.Printf("  Тривалість: %.0f секунд (%.2f хвилин)\n", duration, duration/60)
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nВведіть першу адресу або місце (наприклад, 'м. Київ, вул. Хрещатик'):")
	address1 := readLine(reader)

	fmt.Println("Введіть другу адресу або місце:")
	address2 := readLine(reader)

	fmt.Println("\nОбробка першої адреси...")
	location1, err := geocodeAddress(address1, MAPBOX_ACCESS_TOKEN)
	if err != nil {
		fmt.Printf("Помилка геокодування першої адреси: %v\n\n\n", err)
		main()
	}
	printLocation("Точка 1", location1)

	fmt.Println("\nОбробка другої адреси...")
	location2, err := geocodeAddress(address2, MAPBOX_ACCESS_TOKEN)
	if err != nil {
		fmt.Printf("Помилка геокодування другої адреси: %v\n\n\n", err)
		main()
	}
	printLocation("Точка 2", location2)

	fmt.Println("\nОбчислення маршруту...")
	distance, duration, err := getDistance(location1, location2, MAPBOX_ACCESS_TOKEN)
	if err != nil {
		fmt.Printf("Помилка отримання відстані: %v\n\n\n", err)
		main()
	}
	printDistance(distance, duration)

	fmt.Println("\n=== Робота завершена успішно ===\n\n\n")
	main()
}
