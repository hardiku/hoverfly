package matching

import (
	"github.com/SpectoLabs/hoverfly/core/cache"
	"net/http"
	log "github.com/Sirupsen/logrus"
	"github.com/SpectoLabs/hoverfly/core/models"
)

type RequestMatcher struct {
	RequestCache	cache.Cache
	TemplateStore	RequestTemplateStore
	Webserver	*bool

}

// getResponse returns stored response from cache
func (this *RequestMatcher) GetResponse(req *models.RequestDetails) (*models.ResponseDetails, *MatchingError) {

	var key string

	if *this.Webserver {
		key = req.HashWithoutHost()
	} else {
		key = req.Hash()
	}

	payloadBts, err := this.RequestCache.Get([]byte(key))

	if err != nil {
		log.WithFields(log.Fields{
			"key":         key,
			"error":       err.Error(),
			"query":       req.Query,
			"path":        req.Path,
			"destination": req.Destination,
			"method":      req.Method,
		}).Warn("Failed to retrieve response from cache")

		response, err := this.TemplateStore.GetResponse(*req, *this.Webserver)
		if err != nil {
			log.WithFields(log.Fields{
				"key":         key,
				"error":       err.Error(),
				"query":       req.Query,
				"path":        req.Path,
				"destination": req.Destination,
				"method":      req.Method,
			}).Warn("Failed to find matching request template from template store")

			return nil, &MatchingError{
				StatusCode: 412,
				Description: "Could not find recorded request, please record it first!",
			}
		}
		log.WithFields(log.Fields{
			"key":         key,
			"query":       req.Query,
			"path":        req.Path,
			"destination": req.Destination,
			"method":      req.Method,
		}).Info("Found template matching request from template store")
		return response, nil
	}

	// getting cache response
	payload, err := models.NewPayloadFromBytes(payloadBts)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
			"value": string(payloadBts),
			"key":   key,
		}).Error("Failed to decode payload")
		return nil, &MatchingError{
			StatusCode: 500,
			Description: "Failed to decode payload",
		}
	}

	log.WithFields(log.Fields{
		"key":         key,
		"path":        req.Path,
		"rawQuery":    req.Query,
		"method":      req.Method,
		"destination": req.Destination,
		"status":      payload.Response.Status,
	}).Info("Payload found from cache")

	return &payload.Response, nil
}

func (this *RequestMatcher) SavePayload(payload *models.Payload) (error) {
	var key string

	if *this.Webserver {
		key = payload.IdWithoutHost()
	} else {
		key = payload.Id()
	}

	log.WithFields(log.Fields{
		"path":          payload.Request.Path,
		"rawQuery":      payload.Request.Query,
		"requestMethod": payload.Request.Method,
		"bodyLen":       len(payload.Request.Body),
		"destination":   payload.Request.Destination,
		"hashKey":       key,
	}).Debug("Capturing")

	payloadBytes, err := payload.Encode()

	if err != nil {
		return err
	}

	return this.RequestCache.Set([]byte(key), payloadBytes)
}

type MatchingError struct {
	StatusCode int
	Description string
}

func (this MatchingError) Error() (string) {
	return this.Description
}

// getRequestFingerprint returns request hash
func GetRequestFingerprint(req *http.Request, requestBody []byte, webserver bool) string {
	var r models.RequestDetails

	r = models.RequestDetails{
		Path:        req.URL.Path,
		Method:      req.Method,
		Destination: req.Host,
		Query:       req.URL.RawQuery,
		Body:        string(requestBody),
		Headers:     req.Header,
	}

	if webserver {
		return r.HashWithoutHost()
	}

	return r.Hash()
}