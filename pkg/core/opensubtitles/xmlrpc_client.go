package opensubtitles

import (
	"fmt"

	"github.com/kolo/xmlrpc"
)

const (
	xmlRpcUrl = "https://api.opensubtitles.org:443/xml-rpc" // Use HTTPS
	// UserAgent is already defined in client.go, we can reuse it.
)

// XmlRpcClient manages communication with the OpenSubtitles XML-RPC API.
type XmlRpcClient struct {
	client *xmlrpc.Client
	token  string // Store the login token
	// Add userAgent if needed, or pass it in methods
}

// NewXmlRpcClient creates a new OpenSubtitles XML-RPC API client.
func NewXmlRpcClient() (*XmlRpcClient, error) {
	// TODO: Consider adding User-Agent header customization if library supports it.
	// The kolo/xmlrpc library doesn't seem to have easy header customization.
	client, err := xmlrpc.NewClient(xmlRpcUrl, nil) // nil transport uses DefaultTransport
	if err != nil {
		return nil, fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	return &XmlRpcClient{client: client}, nil
}

// --- XML-RPC Methods ---

// XmlRpcLoginResponse represents the expected structure from the LogIn method.
// Based on typical XML-RPC responses and JS usage.
type XmlRpcLoginResponse struct {
	Token   string  `xmlrpc:"token"`
	Status  string  `xmlrpc:"status"` // e.g., "200 OK"
	Seconds float64 `xmlrpc:"seconds"`
}

// Login performs authentication using the XML-RPC LogIn method.
func (c *XmlRpcClient) Login(username, password, language, userAgent string) error {
	args := []interface{}{username, password, language, userAgent}
	var result XmlRpcLoginResponse

	err := c.client.Call("LogIn", args, &result)
	if err != nil {
		return fmt.Errorf("XML-RPC LogIn call failed: %w", err)
	}

	// Check status string for success (e.g., contains "200 OK")
	if result.Status == "" || result.Status[:3] != "200" {
		return fmt.Errorf("XML-RPC LogIn failed with status: %s", result.Status)
	}

	if result.Token == "" {
		return fmt.Errorf("XML-RPC LogIn succeeded but returned empty token (Status: %s)", result.Status)
	}

	c.token = result.Token
	return nil
}

// Logout performs logout using the XML-RPC LogOut method.
func (c *XmlRpcClient) Logout() error {
	if c.token == "" {
		return fmt.Errorf("cannot logout: not logged in")
	}
	args := []interface{}{c.token}
	var result map[string]interface{} // Logout response is often simple status

	err := c.client.Call("LogOut", args, &result)
	if err != nil {
		// XML-RPC library might return error for invalid token, etc.
		// Logout might succeed even if token was bad server-side.
		// Always clear local token.
		c.token = ""
		return fmt.Errorf("XML-RPC LogOut call failed: %w", err)
	}

	// Check status if possible (structure may vary)
	if status, ok := result["status"]; ok {
		if statusStr, ok := status.(string); ok && statusStr[:3] != "200" {
			c.token = "" // Clear token even on logical failure
			return fmt.Errorf("XML-RPC LogOut failed with status: %s", statusStr)
		}
	}

	c.token = ""
	return nil
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
	Status      string                        `xmlrpc:"status"`
	AlreadyInDB int                           `xmlrpc:"alreadyindb"` // 0 or 1
	Data        []XmlRpcTryUploadResponseData `xmlrpc:"data"`        // Expecting an array
	Seconds     float64                       `xmlrpc:"seconds"`
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

// TryUploadSubtitles calls the XML-RPC method to check if a subtitle exists
// and gather potential metadata before a full upload.
func (c *XmlRpcClient) TryUploadSubtitles(params XmlRpcTryUploadParams) (*XmlRpcTryUploadResponse, error) {
	if c.token == "" {
		return nil, fmt.Errorf("cannot call TryUploadSubtitles: not logged in")
	}

	// The API expects an array containing one map/struct, where the map key is "cd1"
	// and the value is the struct with subtitle/movie details.
	args := []interface{}{
		c.token,
		[]map[string]XmlRpcTryUploadParams{
			{"cd1": params},
		},
	}

	var result XmlRpcTryUploadResponse
	err := c.client.Call("TryUploadSubtitles", args, &result)
	if err != nil {
		return nil, fmt.Errorf("XML-RPC TryUploadSubtitles call failed: %w", err)
	}

	// Check status string for success
	if result.Status == "" || result.Status[:3] != "200" {
		// TODO: Map specific error status codes if known (e.g., 4xx)
		return nil, fmt.Errorf("XML-RPC TryUploadSubtitles failed with status: %s", result.Status)
	}

	// The call succeeded, return the parsed response
	return &result, nil
}

// UploadSubtitles performs the final subtitle upload using the XML-RPC method.
// It requires the prepared parameters including base64 encoded subtitle content.
func (c *XmlRpcClient) UploadSubtitles(params XmlRpcUploadSubtitlesParams) (*XmlRpcUploadSubtitlesResponse, error) {
	if c.token == "" {
		return nil, fmt.Errorf("cannot call UploadSubtitles: not logged in")
	}

	// The API expects the token and an array containing the single structured parameter.
	args := []interface{}{
		c.token,
		[]XmlRpcUploadSubtitlesParams{params},
	}

	var result XmlRpcUploadSubtitlesResponse
	err := c.client.Call("UploadSubtitles", args, &result)
	if err != nil {
		return nil, fmt.Errorf("XML-RPC UploadSubtitles call failed: %w", err)
	}

	// Check status string for success
	if result.Status == "" || result.Status[:3] != "200" {
		// TODO: Map specific error status codes if known (e.g., 4xx like 402, 5xx like 503)
		return nil, fmt.Errorf("XML-RPC UploadSubtitles failed with status: %s", result.Status)
	}

	// JS checks if Data is empty string, let's do the same
	if result.Data == "" {
		return nil, fmt.Errorf("XML-RPC UploadSubtitles succeeded (status %s) but returned empty data URL", result.Status)
	}

	// Upload seems successful
	return &result, nil
}
