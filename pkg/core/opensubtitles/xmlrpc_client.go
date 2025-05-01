package opensubtitles

import (
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/rpc"
	"net/url" // Re-added import

	// "time" // Removed unused import

	"log"

	"github.com/angelospk/osuploadergui/pkg/core/errors"
	xmlrpc "github.com/kolo/xmlrpc"
)

const (
	xmlRpcEndpoint = "https://api.opensubtitles.org:443/xml-rpc"
	// UserAgent is already defined in client.go, we can reuse it.
)

// XmlRpcClient handles communication with the OpenSubtitles XML-RPC API.
type XmlRpcClient struct {
	client   *xmlrpc.Client
	token    string
	loggedIn bool
}

// NewXmlRpcClient creates a new XML-RPC client.
func NewXmlRpcClient() (*XmlRpcClient, error) {
	// Allow insecure connections for potential local testing or specific environments if needed
	// In production, you might want to enforce stricter TLS checks.
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Be cautious with this in production
	}
	// httpClient := &http.Client{Transport: tr} // Removed unused variable

	// Create the kolo/xmlrpc client
	client, err := xmlrpc.NewClient(xmlRpcEndpoint, tr) // Pass tr (RoundTripper) instead of httpClient
	if err != nil {
		return nil, fmt.Errorf("error creating XML-RPC client: %w", err)
	}

	return &XmlRpcClient{
		client:   client,
		loggedIn: false,
	}, nil
}

// Login authenticates the user via XML-RPC and stores the token.
func (c *XmlRpcClient) Login(username, password, language, userAgent string) error {
	var result XmlRpcLoginResponse
	// Use positional arguments for kolo/xmlrpc
	err := c.client.Call("LogIn", []interface{}{username, password, language, userAgent}, &result)
	if err != nil {
		// Check for standard net/rpc errors first
		if err == rpc.ErrShutdown {
			return fmt.Errorf("xmlrpc login connection shutdown: %w", err)
		}
		// Attempt to interpret as kolo/xmlrpc specific error if possible,
		// otherwise, return a generic error.
		// Note: kolo/xmlrpc might not expose detailed error types easily.
		return fmt.Errorf("xmlrpc login call failed: %w", err)
	}

	if result.Status != "200 OK" {
		// Attempt to map known error statuses
		switch result.Status {
		case "401 Unauthorized":
			return errors.ErrUnauthorized
		case "414 Unknown User Agent":
			return fmt.Errorf("xmlrpc login failed: %s (provide a valid UserAgent)", result.Status)
		// Add other known status code mappings here
		default:
			return fmt.Errorf("xmlrpc login failed with status: %s", result.Status)
		}
	}

	c.token = result.Token
	c.loggedIn = true
	fmt.Printf("XML-RPC Login successful. Token: %s\n", c.token) // Added for debugging visibility
	return nil
}

// Logout invalidates the user's session token via XML-RPC.
func (c *XmlRpcClient) Logout() error {
	if !c.loggedIn || c.token == "" {
		return errors.ErrNotLoggedIn
	}
	var result XmlRpcStatusResponse
	// Use positional arguments for kolo/xmlrpc
	err := c.client.Call("LogOut", []interface{}{c.token}, &result)
	if err != nil {
		return fmt.Errorf("xmlrpc logout call failed: %w", err)
	}

	if result.Status != "200 OK" {
		// Handle potential logout errors if the API defines specific ones
		return fmt.Errorf("xmlrpc logout failed with status: %s", result.Status)
	}

	c.token = ""
	c.loggedIn = false
	fmt.Println("XML-RPC Logout successful.") // Added for debugging visibility
	return nil
}

// --- XML-RPC Methods ---

// XmlRpcLoginResponse represents the expected structure from the LogIn method.
// Based on typical XML-RPC responses and JS usage.
type XmlRpcLoginResponse struct {
	Token   string  `xmlrpc:"token"`
	Status  string  `xmlrpc:"status"` // e.g., "200 OK"
	Seconds float64 `xmlrpc:"seconds"`
}

// XmlRpcStatusResponse is a generic response containing just status and time.
type XmlRpcStatusResponse struct {
	Status  string  `xmlrpc:"status"`
	Seconds float64 `xmlrpc:"seconds"`
}

// --- TryUploadSubtitles Structs ---

// XmlRpcTryUploadParams represents the parameters sent within the 'cd1' structure
// for the TryUploadSubtitles XML-RPC method.
// Field names match the JS code analysis.
// All fields seem to be sent as strings, except booleans as "1" or "0".
type XmlRpcTryUploadParams struct {
	SubHash              string `xmlrpc:"subhash"`
	SubFilename          string `xmlrpc:"subfilename"`
	MovieHash            string `xmlrpc:"moviehash"`
	MovieByteSize        string `xmlrpc:"moviebytesize"`
	MovieFilename        string `xmlrpc:"moviefilename"`
	IDMovieImdb          string `xmlrpc:"idmovieimdb,omitempty"`          // Optional
	SubLanguageID        string `xmlrpc:"sublanguageid,omitempty"`        // Optional
	MovieFPS             string `xmlrpc:"moviefps,omitempty"`             // Optional
	MovieFrames          string `xmlrpc:"movieframes,omitempty"`          // Optional
	MovieTimeMS          string `xmlrpc:"movietimems,omitempty"`          // Optional
	SubAuthorComment     string `xmlrpc:"subauthorcomment,omitempty"`     // Optional
	SubTranslator        string `xmlrpc:"subtranslator,omitempty"`        // Optional
	MovieReleaseName     string `xmlrpc:"moviereleasename,omitempty"`     // Optional
	MovieAka             string `xmlrpc:"movieaka,omitempty"`             // Optional
	HearingImpaired      string `xmlrpc:"hearingimpaired,omitempty"`      // Optional ("1" or "0")
	HighDefinition       string `xmlrpc:"highdefinition,omitempty"`       // Optional ("1" or "0")
	AutomaticTranslation string `xmlrpc:"automatictranslation,omitempty"` // Optional ("1" or "0")
	ForeignPartsOnly     string `xmlrpc:"foreignpartsonly,omitempty"`     // Optional ("1" or "0")
}

// XmlRpcTryUploadResponseData represents the structure within the 'data' array
// of the TryUploadSubtitles response.
type XmlRpcTryUploadResponseData struct {
	IDMovieImdb string `xmlrpc:"IDMovieImdb"` // String based on JS usage
	// Add other fields if the API returns more useful info here
}

// XmlRpcTryUploadResponse represents the expected structure from the TryUploadSubtitles method.
type XmlRpcTryUploadResponse struct {
	Status       string      `xmlrpc:"status"`
	Data         bool        `xmlrpc:"data"` // Indicates if upload should proceed (true) or is duplicate (false)
	RawData      interface{} // Holds the raw 'data' field for further processing (array/object)
	AlreadyInDB  int         `xmlrpc:"alreadyindb"` // Usually 1 if duplicate, 0 otherwise
	Seconds      float64     `xmlrpc:"seconds"`
	SubActualCDN string      `xmlrpc:"subactualcdn"` // Added field based on potential responses
}

// --- End TryUploadSubtitles Structs ---

// --- UploadSubtitles Structs ---

// XmlRpcUploadSubtitlesBaseInfo holds the base metadata for UploadSubtitles.
// Fields derived from JS arrangeUploadData function. Sent as strings.
type XmlRpcUploadSubtitlesBaseInfo struct {
	IDMovieImdb          string `xmlrpc:"idmovieimdb,omitempty"`
	MovieReleaseName     string `xmlrpc:"moviereleasename,omitempty"`
	MovieAka             string `xmlrpc:"movieaka,omitempty"`
	SubLanguageID        string `xmlrpc:"sublanguageid,omitempty"`
	SubAuthorComment     string `xmlrpc:"subauthorcomment,omitempty"`
	HearingImpaired      string `xmlrpc:"hearingimpaired,omitempty"`      // "1" or "0"
	HighDefinition       string `xmlrpc:"highdefinition,omitempty"`       // "1" or "0"
	AutomaticTranslation string `xmlrpc:"automatictranslation,omitempty"` // "1" or "0"
	SubTranslator        string `xmlrpc:"subtranslator,omitempty"`
	ForeignPartsOnly     string `xmlrpc:"foreignpartsonly,omitempty"` // "1" or "0"
}

// XmlRpcUploadSubtitlesCD holds the subtitle file details for UploadSubtitles.
// Fields derived from JS arrangeUploadData function. Sent as strings.
type XmlRpcUploadSubtitlesCD struct {
	SubHash       string `xmlrpc:"subhash"`
	SubFilename   string `xmlrpc:"subfilename"`
	SubContent    string `xmlrpc:"subcontent"` // Base64 encoded content
	MovieByteSize string `xmlrpc:"moviebytesize,omitempty"`
	MovieHash     string `xmlrpc:"moviehash,omitempty"`
	MovieFilename string `xmlrpc:"moviefilename,omitempty"`
	MovieFPS      string `xmlrpc:"moviefps,omitempty"`
	MovieFrames   string `xmlrpc:"movieframes,omitempty"`
	MovieTimeMS   string `xmlrpc:"movietimems,omitempty"`
}

// XmlRpcUploadSubtitlesParams is the top-level structure sent to UploadSubtitles.
type XmlRpcUploadSubtitlesParams struct {
	BaseInfo XmlRpcUploadSubtitlesBaseInfo `xmlrpc:"baseinfo"`
	CD1      XmlRpcUploadSubtitlesCD       `xmlrpc:"cd1"` // Assumes only one CD/file per upload
}

// XmlRpcUploadSubtitlesResponse represents the structure from UploadSubtitles.
// Based on JS checks (status, data as string URL).
type XmlRpcUploadSubtitlesResponse struct {
	Status  string  `xmlrpc:"status"`
	Data    string  `xmlrpc:"data"` // Expected to be the subtitle page URL string
	Seconds float64 `xmlrpc:"seconds"`
}

// --- End UploadSubtitles Structs ---

// TryUploadSubtitles performs the first step of the upload process.
func (c *XmlRpcClient) TryUploadSubtitles(params XmlRpcTryUploadParams) (*XmlRpcTryUploadResponse, error) {
	if !c.loggedIn || c.token == "" {
		return nil, errors.ErrNotLoggedIn
	}

	// Prepare the complex structure expected by the API
	cdMap := make(map[string]interface{})
	cdMap["subhash"] = params.SubHash
	cdMap["subfilename"] = params.SubFilename
	cdMap["moviehash"] = params.MovieHash
	cdMap["moviebytesize"] = params.MovieByteSize // Already a string
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

	// Use xmlrpc.RawValue to decode the response flexibly
	var rawResp interface{}
	err := c.client.Call("TryUploadSubtitles", args, &rawResp)
	if err != nil {
		return nil, fmt.Errorf("xmlrpc TryUploadSubtitles call failed: %w", err)
	}
	log.Printf("[DEBUG] Raw TryUploadSubtitles response: %+v (type: %T)", rawResp, rawResp)

	// Try to interpret the response as a map (struct) or bool
	switch v := rawResp.(type) {
	case map[string]interface{}:
		var result XmlRpcTryUploadResponse
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
		if data, ok := v["data"]; ok {
			result.RawData = data // Save the raw data for further processing
			// JS logic: if alreadyindb==1, treat as duplicate and return early
			if result.AlreadyInDB == 1 {
				result.Data = false
				return &result, errors.ErrUploadDuplicate
			}
			// If alreadyindb==0, proceed and return the parsed response (including the data array/object)
			// JS code expects to parse/use this data in the next step
			result.Data = true
			return &result, nil
		}
		// Defensive: if no data field, treat as error
		return nil, fmt.Errorf("xmlrpc TryUploadSubtitles missing 'data' field")
	case bool:
		// If the response is just a bool, treat as Data field
		if v {
			return &XmlRpcTryUploadResponse{Status: "200 OK", Data: true, RawData: v, AlreadyInDB: 0}, nil
		}
		return &XmlRpcTryUploadResponse{Status: "200 OK", Data: false, RawData: v, AlreadyInDB: 1}, errors.ErrUploadDuplicate
	default:
		return nil, fmt.Errorf("unexpected TryUploadSubtitles response type: %T (%v)", rawResp, rawResp)
	}
}

// UploadSubtitles performs the second step, uploading the actual subtitle file content.
func (c *XmlRpcClient) UploadSubtitles(params XmlRpcUploadSubtitlesParams) (*XmlRpcUploadSubtitlesResponse, error) {
	if !c.loggedIn || c.token == "" {
		return nil, errors.ErrNotLoggedIn
	}

	// Prepare the structure for UploadSubtitles
	// This involves base64 encoded, gzipped subtitle content.
	subContentMap := make(map[string]interface{})
	subContentMap["idsubtitlefile"] = params.BaseInfo.IDMovieImdb // From TryUpload response
	subContentMap["subcontent"] = params.CD1.SubContent           // Base64(Gzip(file_content))

	// --- BEGIN DEBUG LOGGING ---
	maskedToken := ""
	if len(c.token) > 8 {
		maskedToken = c.token[:4] + "..." + c.token[len(c.token)-4:]
	} else {
		maskedToken = c.token
	}
	base64Len := len(params.CD1.SubContent)
	base64Hash := ""
	if base64Len > 0 {
		base64Hash = fmt.Sprintf("%x", md5Sum([]byte(params.CD1.SubContent)))
	}
	log.Printf("[DEBUG] UploadSubtitles request: token=%s, idmovieimdb=%q, subhash=%q, subfilename=%q, base64len=%d, base64md5=%s, moviehash=%q, moviebytesize=%q, moviefilename=%q, moviefps=%q, movieframes=%q, movietimems=%q, hearingimpaired=%q, highdefinition=%q, automatictranslation=%q, foreignpartsonly=%q, subtranslator=%q, subauthorcomment=%q, moviereleasename=%q, movieaka=%q", maskedToken, params.BaseInfo.IDMovieImdb, params.CD1.SubHash, params.CD1.SubFilename, base64Len, base64Hash, params.CD1.MovieHash, params.CD1.MovieByteSize, params.CD1.MovieFilename, params.CD1.MovieFPS, params.CD1.MovieFrames, params.CD1.MovieTimeMS, params.BaseInfo.HearingImpaired, params.BaseInfo.HighDefinition, params.BaseInfo.AutomaticTranslation, params.BaseInfo.ForeignPartsOnly, params.BaseInfo.SubTranslator, params.BaseInfo.SubAuthorComment, params.BaseInfo.MovieReleaseName, params.BaseInfo.MovieAka)
	// --- END DEBUG LOGGING ---

	args := []interface{}{
		c.token,
		[]interface{}{ // Array of subtitle contents
			subContentMap,
		},
		// No third 'baseinfo' argument for UploadSubtitles according to docs/common practice
	}

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

	log.Printf("[DEBUG] Raw UploadSubtitles response: %+v (type: %T)", rawResp, rawResp)

	// Accept both map[string]interface{} and direct string (URL) as 'data'
	switch v := rawResp.(type) {
	case map[string]interface{}:
		var result XmlRpcUploadSubtitlesResponse
		if status, ok := v["status"].(string); ok {
			result.Status = status
		}
		if data, ok := v["data"]; ok {
			switch dataTyped := data.(type) {
			case string:
				// This is the URL to the uploaded subtitle page
				result.Data = dataTyped
				log.Printf("[DEBUG] UploadSubtitles: data is URL string: %s", dataTyped)
			// Optionally, parse the URL for user feedback (TODO)
			default:
				log.Printf("[DEBUG] UploadSubtitles: data is unexpected type: %T", dataTyped)
			}
		}
		if seconds, ok := v["seconds"].(float64); ok {
			result.Seconds = seconds
		}
		if alreadyInDB, ok := v["alreadyindb"].(int); ok {
			// Not always present, but handle if so
			_ = alreadyInDB // Not used yet
		}
		if result.Status != "200 OK" {
			return nil, fmt.Errorf("xmlrpc UploadSubtitles failed with status: %s", result.Status)
		}
		return &result, nil
	default:
		return nil, fmt.Errorf("unexpected UploadSubtitles response type: %T (%v)", rawResp, rawResp)
	}
}

// md5Sum returns the MD5 hash of the input bytes.
func md5Sum(data []byte) []byte {
	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}

// Close closes the underlying XML-RPC client connection.
func (c *XmlRpcClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
