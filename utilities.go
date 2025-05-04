package opensubtitles

import "context"

// Methods related to utility endpoints (Guessit)

// Guessit attempts to parse structured information (title, year, season, etc.)
// from a filename using the OpenSubtitles guessit utility.
func (c *Client) Guessit(ctx context.Context, params GuessitParams) (*GuessitResponse, error) {
	var response GuessitResponse
	// Params struct has `url` tags for query string encoding
	err := c.httpClient.Get(ctx, "/utilities/guessit", params, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
