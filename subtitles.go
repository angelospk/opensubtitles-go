package opensubtitles

import "context"

// Methods related to subtitles (Search, Download)

// SearchSubtitles searches for subtitles based on various criteria.
func (c *Client) SearchSubtitles(ctx context.Context, params SearchSubtitlesParams) (*SearchSubtitlesResponse, error) {
	var response SearchSubtitlesResponse
	// Params struct already has `url` tags for query string encoding
	err := c.httpClient.Get(ctx, "/subtitles", params, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// Download requests a download link for a specific subtitle file.
// Requires authentication.
func (c *Client) Download(ctx context.Context, params DownloadRequest) (*DownloadResponse, error) {
	// Authentication token is added automatically by the httpClient if available.
	var response DownloadResponse
	err := c.httpClient.Post(ctx, "/download", params, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// TODO: Implement DownloadSubtitle
// func (c *Client) DownloadSubtitle(ctx context.Context, params DownloadRequest) (*DownloadResponse, error) { ... }
