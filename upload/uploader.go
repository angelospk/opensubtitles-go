package upload

// This package will contain the logic for uploading subtitles via XML-RPC.

// TODO: Refactor the existing XML-RPC client logic (xmlrpc_client.go)
// into this package. Define an Uploader interface and implementation.

// Example Interface:
// type Uploader interface {
// 	 Login(username, password, userAgent string) error
// 	 Logout() error
// 	 UploadSubtitle(params UploadParams) (*UploadResponse, error)
// }

// type xmlRpcUploader struct {
// 	 // ... xmlrpc client instance, token, etc.
// }

// func NewXmlRpcUploader() (Uploader, error) { ... }

// func (u *xmlRpcUploader) Login(...) error { ... }
// func (u *xmlRpcUploader) Logout() error { ... }
// func (u *xmlRpcUploader) UploadSubtitle(...) (*UploadResponse, error) { ... }

// Define necessary structs for XML-RPC upload parameters and responses here,
// potentially reusing/adapting from the old xmlrpc_client.go

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"net/url"
	"time"

	xmlrpc "github.com/kolo/xmlrpc"
)

const (
	xmlRpcEndpoint = "https://api.opensubtitles.org:443/xml-rpc"
	// UserAgent is already defined in client.go, we can reuse it.
)

// --- Public Interface & Structs ---

// Uploader defines the interface for XML-RPC upload operations.
type Uploader interface {
	// Login authenticates using username and MD5 HASHED password.
	Login(username, md5Password, language, userAgent string) error
	Logout() error
	// Upload performs the complete two-step subtitle upload process.
	// It takes user intent, prepares parameters, calls TryUpload and UploadSubtitles.
	// Returns the URL of the uploaded subtitle on success.
	Upload(intent UserUploadIntent) (string, error)
	Close() error // Add Close method to the interface
}

// Errors returned by the upload package
var (
	ErrNotLoggedIn     = errors.New("uploader not logged in")
	ErrUploadDuplicate = errors.New("upload failed: subtitle already in database")
	ErrUnauthorized    = errors.New("xmlrpc login failed: 401 Unauthorized")
)

// --- Implementation ---

// xmlRpcClient handles communication with the OpenSubtitles XML-RPC API.
// Renamed from XmlRpcClient (unexported)
type xmlRpcClient struct {
	client   *xmlrpc.Client
	token    string
	loggedIn bool
}

// Ensure xmlRpcClient implements Uploader.
var _ Uploader = (*xmlRpcClient)(nil)

// NewXmlRpcUploader creates a new XML-RPC uploader client.
// Renamed from NewXmlRpcClient
func NewXmlRpcUploader() (Uploader, error) {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	httpClient := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	client, err := xmlrpc.NewClient(xmlRpcEndpoint, httpClient.Transport)
	if err != nil {
		return nil, fmt.Errorf("error creating XML-RPC client: %w", err)
	}

	return &xmlRpcClient{
		client:   client,
		loggedIn: false,
	}, nil
}

// Login authenticates the user via XML-RPC and stores the token.
func (c *xmlRpcClient) Login(username, password, language, userAgent string) error {
	var result xmlRpcLoginResponse // Use unexported struct
	err := c.client.Call("LogIn", []interface{}{username, password, language, userAgent}, &result)
	if err != nil {
		if err == rpc.ErrShutdown {
			return fmt.Errorf("xmlrpc login connection shutdown: %w", err)
		}
		return fmt.Errorf("xmlrpc login call failed: %w", err)
	}

	if result.Status != "200 OK" {
		switch result.Status {
		case "401 Unauthorized":
			return ErrUnauthorized // Use defined error
		case "414 Unknown User Agent":
			return fmt.Errorf("xmlrpc login failed: %s (provide a valid UserAgent)", result.Status)
		default:
			return fmt.Errorf("xmlrpc login failed with status: %s", result.Status)
		}
	}

	c.token = result.Token
	c.loggedIn = true
	fmt.Printf("XML-RPC Login successful. Token: %s...\n", c.token[:4]) // Shorten token log
	return nil
}

// Logout invalidates the user's session token via XML-RPC.
func (c *xmlRpcClient) Logout() error {
	if !c.loggedIn || c.token == "" {
		return ErrNotLoggedIn // Use defined error
	}
	var result xmlRpcStatusResponse // Use unexported struct
	err := c.client.Call("LogOut", []interface{}{c.token}, &result)
	if err != nil {
		return fmt.Errorf("xmlrpc logout call failed: %w", err)
	}

	if result.Status != "200 OK" {
		return fmt.Errorf("xmlrpc logout failed with status: %s", result.Status)
	}

	c.token = ""
	c.loggedIn = false
	fmt.Println("XML-RPC Logout successful.")
	return nil
}

// Upload performs the full two-step upload process.
func (c *xmlRpcClient) Upload(intent UserUploadIntent) (string, error) {
	if !c.loggedIn || c.token == "" {
		return "", ErrNotLoggedIn
	}

	// 1. Prepare TryUpload parameters
	log.Println("Preparing TryUpload parameters...")
	tryParams, err := PrepareTryUploadParams(intent) // From helpers.go
	if err != nil {
		return "", fmt.Errorf("error preparing TryUpload params: %w", err)
	}
	log.Printf("[DEBUG] TryUpload Params: %+v\n", tryParams)

	// Optional: Modify filename for uniqueness? (Consider if needed)
	// uniqueSubFilename := fmt.Sprintf("%s_%d.srt", filepath.Base(intent.SubtitleFilePath), time.Now().UnixNano())
	// tryParams.SubFilename = uniqueSubFilename

	// 2. Call TryUploadSubtitles
	log.Println("Calling TryUploadSubtitles...")
	tryResponse, err := c.tryUploadSubtitles(tryParams) // Call internal method
	if err != nil {
		if errors.Is(err, ErrUploadDuplicate) {
			log.Println("TryUploadSubtitles indicates duplicate.")
			return "", ErrUploadDuplicate
		}
		return "", fmt.Errorf("TryUploadSubtitles failed: %w", err)
	}
	log.Printf("TryUploadSubtitles response: Status='%s', Data=%v, AlreadyInDB=%d", tryResponse.Status, tryResponse.Data, tryResponse.AlreadyInDB)

	// 3. Check if TryUpload response indicates we should proceed
	if !tryResponse.Data {
		log.Println("TryUpload response indicates duplicate or issue (Data=false). Skipping final upload.")
		return "", ErrUploadDuplicate // Treat non-proceed as duplicate error for simplicity
	}

	// 4. Prepare UploadSubtitles parameters
	log.Println("Preparing UploadSubtitles parameters...")
	uploadParams, err := PrepareUploadSubtitlesParams(tryParams, intent.SubtitleFilePath) // From helpers.go
	if err != nil {
		return "", fmt.Errorf("error preparing UploadSubtitles params: %w", err)
	}
	// fmt.Printf("[DEBUG] UploadSubtitles Params: %+v\n", uploadParams) // Keep commented unless needed

	// 5. Call UploadSubtitles
	log.Println("Calling UploadSubtitles...")
	uploadResp, err := c.uploadSubtitles(uploadParams) // Call internal method
	if err != nil {
		return "", fmt.Errorf("UploadSubtitles failed: %w", err)
	}
	log.Printf("UploadSubtitles successful! Status: %s, URL: %s", uploadResp.Status, uploadResp.Data)

	return uploadResp.Data, nil // Return the subtitle URL
}

// Close closes the underlying XML-RPC client connection.
func (c *xmlRpcClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// --- Internal XML-RPC Methods & Structs (unexported) ---

// xmlRpcLoginResponse represents the expected structure from the LogIn method.
type xmlRpcLoginResponse struct {
	Token   string  `xmlrpc:"token"`
	Status  string  `xmlrpc:"status"`
	Seconds float64 `xmlrpc:"seconds"`
}

// xmlRpcStatusResponse is a generic response containing just status and time.
type xmlRpcStatusResponse struct {
	Status  string  `xmlrpc:"status"`
	Seconds float64 `xmlrpc:"seconds"`
}

// xmlRpcTryUploadResponse represents the expected structure from the TryUploadSubtitles method.
type xmlRpcTryUploadResponse struct {
	Status       string      `xmlrpc:"status"`
	Data         bool        `xmlrpc:"data"`
	RawData      interface{} `xmlrpc:"-"` // Ignore raw data for now
	AlreadyInDB  int         `xmlrpc:"alreadyindb"`
	Seconds      float64     `xmlrpc:"seconds"`
	SubActualCDN string      `xmlrpc:"subactualcdn"`
}

// xmlRpcUploadSubtitlesResponse represents the structure from UploadSubtitles.
type xmlRpcUploadSubtitlesResponse struct {
	Status    string  `xmlrpc:"status"`
	Data      string  `xmlrpc:"data"`
	Subtitles bool    `xmlrpc:"subtitles"`
	Seconds   float64 `xmlrpc:"seconds"`
}

// --- Internal Method Implementations ---
// Renamed TryUploadSubtitles to unexported
func (c *xmlRpcClient) tryUploadSubtitles(params xmlRpcTryUploadParams) (*xmlRpcTryUploadResponse, error) {

	// Prepare the complex structure expected by the API
	cdMap := make(map[string]interface{})
	cdMap["subhash"] = params.SubHash
	cdMap["subfilename"] = params.SubFilename
	cdMap["moviehash"] = params.MovieHash
	cdMap["moviebytesize"] = params.MovieByteSize
	cdMap["moviefilename"] = params.MovieFilename
	if params.IDMovieImdb != "" {
		cdMap["imdbid"] = params.IDMovieImdb
	}
	if params.MovieFPS != "" {
		cdMap["moviefps"] = params.MovieFPS
	}
	if params.MovieTimeMS != "" {
		cdMap["movietimems"] = params.MovieTimeMS
	}
	if params.SubAuthorComment != "" {
		cdMap["subauthorcomment"] = params.SubAuthorComment
	}
	if params.SubTranslator != "" {
		cdMap["subtranslator"] = params.SubTranslator
	}
	if params.MovieReleaseName != "" {
		cdMap["moviereleasename"] = params.MovieReleaseName
	}
	if params.MovieAka != "" {
		cdMap["movieaka"] = params.MovieAka
	}
	if params.HearingImpaired != "" {
		cdMap["hearingimpaired"] = params.HearingImpaired
	}
	if params.HighDefinition != "" {
		cdMap["highdefinition"] = params.HighDefinition
	}
	if params.AutomaticTranslation != "" {
		cdMap["automatictranslation"] = params.AutomaticTranslation
	}
	if params.ForeignPartsOnly != "" {
		cdMap["foreignpartsonly"] = params.ForeignPartsOnly
	}

	baseInfoMap := make(map[string]interface{})
	if params.IDMovieImdb != "" {
		baseInfoMap["idmovieimdb"] = params.IDMovieImdb
	}
	if params.SubLanguageID != "" {
		baseInfoMap["sublanguageid"] = params.SubLanguageID
	}
	if params.MovieReleaseName != "" {
		baseInfoMap["moviereleasename"] = params.MovieReleaseName
	}
	if params.MovieAka != "" {
		baseInfoMap["movieaka"] = params.MovieAka
	}
	if params.SubAuthorComment != "" {
		baseInfoMap["subauthorcomment"] = params.SubAuthorComment
	}
	if params.SubTranslator != "" {
		baseInfoMap["subtranslator"] = params.SubTranslator
	}
	if params.ForeignPartsOnly != "" {
		baseInfoMap["foreignpartsonly"] = params.ForeignPartsOnly
	}

	args := []interface{}{
		c.token,
		[]interface{}{cdMap},
		baseInfoMap,
	}

	var rawResp interface{}
	err := c.client.Call("TryUploadSubtitles", args, &rawResp)
	if err != nil {
		return nil, fmt.Errorf("xmlrpc TryUploadSubtitles call failed: %w", err)
	}

	switch v := rawResp.(type) {
	case map[string]interface{}:
		var result xmlRpcTryUploadResponse
		if status, ok := v["status"].(string); ok {
			result.Status = status
		}
		if alreadyInDB, ok := v["alreadyindb"].(int); ok {
			result.AlreadyInDB = alreadyInDB
		} else if alreadyInDBf, ok := v["alreadyindb"].(float64); ok {
			result.AlreadyInDB = int(alreadyInDBf)
		}
		if seconds, ok := v["seconds"].(float64); ok {
			result.Seconds = seconds
		}
		if subActualCDN, ok := v["subactualcdn"].(string); ok {
			result.SubActualCDN = subActualCDN
		}
		if _, dataOK := v["data"]; dataOK {
			// Treat presence of data field and alreadyindb==0 as success
			if result.AlreadyInDB == 1 {
				result.Data = false
				return &result, ErrUploadDuplicate // Use defined error
			} else {
				result.Data = true
				return &result, nil
			}
		}
		return nil, fmt.Errorf("xmlrpc TryUploadSubtitles missing 'data' field or unexpected structure: status %s", result.Status)
	case bool: // Should ideally not happen if alreadyindb is present
		if v {
			return &xmlRpcTryUploadResponse{Status: "200 OK", Data: true, AlreadyInDB: 0}, nil
		}
		return &xmlRpcTryUploadResponse{Status: "200 OK", Data: false, AlreadyInDB: 1}, ErrUploadDuplicate // Use defined error
	default:
		return nil, fmt.Errorf("unexpected TryUploadSubtitles response type: %T (%v)", rawResp, rawResp)
	}
}

// Renamed UploadSubtitles to unexported
func (c *xmlRpcClient) uploadSubtitles(params xmlRpcUploadSubtitlesParams) (*xmlRpcUploadSubtitlesResponse, error) {

	// Assuming we only handle "cd1" for now, extract it - Removed unused var check
	// cd1, ok := params.CDs["cd1"]
	// if !ok {
	// 	 return nil, fmt.Errorf("missing 'cd1' data in UploadSubtitles parameters")
	// }
	if _, ok := params.CDs["cd1"]; !ok { // Just check existence if needed later, but not used now.
		log.Println("[WARN] 'cd1' key missing in upload parameters, but proceedeing.")
		// return nil, fmt.Errorf("missing 'cd1' data in UploadSubtitles parameters")
	}

	// Construct the final nested structure for the XML-RPC call
	callParams := map[string]interface{}{"baseinfo": params.BaseInfo}
	for key, cdData := range params.CDs {
		callParams[key] = cdData
	}

	args := []interface{}{c.token, callParams}

	var rawResp interface{}
	err := c.client.Call("UploadSubtitles", args, &rawResp)
	if err != nil {
		if err == rpc.ErrShutdown {
			return nil, fmt.Errorf("xmlrpc UploadSubtitles connection shutdown: %w", err)
		}
		if urlErr, ok := err.(*url.Error); ok {
			return nil, fmt.Errorf("xmlrpc UploadSubtitles network/url error: %w", urlErr)
		}
		return nil, fmt.Errorf("xmlrpc UploadSubtitles call failed: %w", err)
	}

	switch v := rawResp.(type) {
	case map[string]interface{}:
		var result xmlRpcUploadSubtitlesResponse
		if status, ok := v["status"].(string); ok {
			result.Status = status
		}
		if data, ok := v["data"].(string); ok {
			result.Data = data
		} else if dataRaw, dataOK := v["data"]; dataOK && dataRaw == nil {
			// Handle case where 'data' is present but null (might indicate failure despite 200 OK status text?)
			log.Printf("[WARN] UploadSubtitles received 'data': <nil> with status: %s", result.Status)
		} else {
			log.Printf("[WARN] UploadSubtitles 'data' field missing or not a string: %T (%v)", dataRaw, dataRaw)
		}

		if subtitles, ok := v["subtitles"].(bool); ok {
			result.Subtitles = subtitles
		} else if subtitlesInt, ok := v["subtitles"].(int); ok {
			result.Subtitles = (subtitlesInt != 0)
		}
		if seconds, ok := v["seconds"].(float64); ok {
			result.Seconds = seconds
		}
		if result.Status != "200 OK" {
			log.Printf("[ERROR] UploadSubtitles failed. Status: %s, Raw Response: %+v", result.Status, v)
			return nil, fmt.Errorf("xmlrpc UploadSubtitles failed with status: %s", result.Status)
		}
		// Check if data URL is empty even if status is 200 OK
		if result.Data == "" {
			log.Printf("[WARN] UploadSubtitles status 200 OK but data URL is empty. Raw: %+v", v)
			// Consider returning an error here if an empty URL always means failure
			// return nil, fmt.Errorf("xmlrpc UploadSubtitles status 200 OK but data URL is empty")
		}
		return &result, nil
	default:
		log.Printf("[ERROR] Unexpected UploadSubtitles response type: %T, Value: %+v", rawResp, rawResp)
		return nil, fmt.Errorf("unexpected UploadSubtitles response type: %T (%v)", rawResp, rawResp)
	}
}

// md5Sum returns the MD5 hash of the input bytes.
func md5Sum(data []byte) []byte {
	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}

// boolToString defined in helpers.go

// Structs used by helpers, need to be defined here or accessible
// (Defined globally in helpers.go for now)
// type XmlRpcTryUploadParams struct { ... }
// type XmlRpcUploadSubtitlesBaseInfo struct { ... }
// type XmlRpcUploadSubtitlesCD struct { ... }
// type XmlRpcUploadSubtitlesParams struct { ... }
