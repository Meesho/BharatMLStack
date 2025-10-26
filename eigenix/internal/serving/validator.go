package serving

import (
	"errors"

	"github.com/Meesho/skye-eigenix/internal/client"
)

func ValidateCreateIndexRequest(req *client.CreateIndexRequest) error {
	if req.Name == "" {
		return errors.New("name is required")
	}
	if req.Dimension == 0 {
		return errors.New("dimension is required")
	}
	if req.Space == "" {
		return errors.New("space is required")
	}
	if req.MaxElements == 0 {
		return errors.New("max_elements is required")
	}
	if req.M == 0 {
		return errors.New("m is required")
	}
	if req.EfConstruction == 0 {
		return errors.New("ef_construction is required")
	}
	return nil
}

func ValidateSearchRequest(req *client.SearchRequest) error {
	if req.IndexName == "" {
		return errors.New("index name is required")
	}
	if req.Limit <= 0 {
		return errors.New("limit must be greater than 0")
	}
	if len(req.Vector) == 0 {
		return errors.New("vector is required")
	}

	// Validate SearchParams if provided
	if req.SearchParams != nil {
		if req.SearchParams.UseIvf {
			// If IVF is enabled, centroid count must be specified
			if req.SearchParams.CentCount <= 0 {
				return errors.New("cent_count must be greater than 0 when use_ivf is true")
			}
		}
	}

	return nil
}

func ValidateBatchSearchRequest(req *client.BatchSearchRequest) error {
	if req.IndexName == "" {
		return errors.New("index name is required")
	}
	if req.Limit <= 0 {
		return errors.New("limit must be greater than 0")
	}
	if len(req.Vectors) == 0 {
		return errors.New("vectors are required")
	}

	// Validate SearchParams if provided
	if req.SearchParams != nil {
		if req.SearchParams.UseIvf {
			// If IVF is enabled, centroid count must be specified
			if req.SearchParams.CentCount <= 0 {
				return errors.New("cent_count must be greater than 0 when use_ivf is true")
			}
		}
	}

	return nil
}
