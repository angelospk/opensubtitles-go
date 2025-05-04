package opensubtitles

import "context"

// Methods related to discovery endpoints (Popular, Latest, MostDownloaded)

// DiscoverPopular retrieves popular features (movies/tvshows).
func (c *Client) DiscoverPopular(ctx context.Context, params DiscoverParams) (*DiscoverPopularResponse, error) {
	var response DiscoverPopularResponse
	err := c.httpClient.Get(ctx, "/discover/popular", params, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// DiscoverLatest retrieves the latest added subtitles.
func (c *Client) DiscoverLatest(ctx context.Context, params DiscoverParams) (*DiscoverLatestResponse, error) {
	var response DiscoverLatestResponse
	err := c.httpClient.Get(ctx, "/discover/latest", params, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// DiscoverMostDownloaded retrieves the most downloaded subtitles.
func (c *Client) DiscoverMostDownloaded(ctx context.Context, params DiscoverParams) (*DiscoverMostDownloadedResponse, error) {
	var response DiscoverMostDownloadedResponse
	err := c.httpClient.Get(ctx, "/discover/most_downloaded", params, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
