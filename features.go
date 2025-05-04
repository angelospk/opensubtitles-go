package opensubtitles

import "context"

// Methods related to features (Movies, TV Shows, Episodes)

// SearchFeatures searches for features (movies, tvshows, episodes) based on criteria.
// Note: The response contains a slice of Feature structs where the Attributes field
// is an interface{}. Users will need to inspect the FeatureType within the attributes
// (after potentially unmarshalling the interface{} into a map[string]interface{})
// and then perform a type assertion or unmarshal the attributes into the specific
// type (FeatureMovieAttributes, FeatureTvshowAttributes, FeatureEpisodeAttributes)
// to access type-specific fields.
func (c *Client) SearchFeatures(ctx context.Context, params SearchFeaturesParams) (*SearchFeaturesResponse, error) {
	var response SearchFeaturesResponse
	// Params struct has `url` tags for query string encoding
	err := c.httpClient.Get(ctx, "/features", params, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
