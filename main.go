package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"database/sql"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	sheets "google.golang.org/api/sheets/v4"

	_ "github.com/lib/pq"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile := "./cache.json"
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

type body struct {
	Data struct {
		Range  string     `json:"range"`
		Values [][]string `json:"values"`
	} `json:"data"`
	ValueInputOption string `json:"valueInputOption"`
}

// initiziation function
func init() {
	// setup config
	config()

	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the Info severity or above.
	log.SetLevel(log.DebugLevel)
}

func main() {

	ctx := context.Background()
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(ctx, config)
	sheetsService, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	loaders := viper.GetStringMap("loaders")
	for loader := range loaders {
		log.Infof("Running Loader %s", loader)
		spreadsheetId := viper.GetString("loaders." + loader + ".sheet")
		rangeData := viper.GetString("loaders.ulta.range")

		db := DbConnect()
		rows, _ := db.Query(viper.GetString("loaders." + loader + ".query"))
		defer rows.Close()

		columns, _ := rows.Columns()
		headers := make([]interface{}, len(columns))
		for i := range columns {
			headers[i] = columns[i]
		}
		sheetValues := [][]interface{}{headers}
		count := len(columns)
		values := make([]interface{}, count)
		valuePtrs := make([]interface{}, count)

		for rows.Next() {
			for i := range columns {
				valuePtrs[i] = &values[i]
			}
			var x []interface{}
			switch err := rows.Scan(valuePtrs...); err {
			case sql.ErrNoRows:
				panic("something went wrong..")
			case nil:
				for i, _ := range columns {
					val := values[i]

					b, ok := val.([]byte)
					var v interface{}
					if ok {
						v = string(b)
					} else {
						v = val
					}
					x = append(x, v)
				}
				sheetValues = append(sheetValues, x)
			default:
				panic("row scan failure")
			}
		}

		rb := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
		}
		rb.Data = append(rb.Data, &sheets.ValueRange{
			Range:  rangeData,
			Values: sheetValues,
		})
		_, err = sheetsService.Spreadsheets.Values.BatchUpdate(spreadsheetId, rb).Context(ctx).Do()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Done.")
	}
}

func DbConnect() *sql.DB {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s",
		viper.GetString("database.host"), viper.GetInt("database.port"), viper.GetString("database.user"), viper.GetString("database.password"), viper.GetString("database.dbname"))
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to Postgres DB")
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Unable to connect to Postgres DB")
	}

	log.Debug("You are Successfully connected!")

	return db
}

func config() {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Could not read config file: %s \n", err))
	}
}
