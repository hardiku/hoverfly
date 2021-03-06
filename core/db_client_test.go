package hoverfly

import (
	. "github.com/onsi/gomega"
	"encoding/json"
	"fmt"
	"github.com/SpectoLabs/hoverfly/core/cache"
	"github.com/SpectoLabs/hoverfly/core/models"
	"net/http"
	"testing"
)

func TestSetKey(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()
	defer dbClient.RequestCache.DeleteData()

	k := []byte("randomkeyhere")
	v := []byte("value")

	err := dbClient.RequestCache.Set(k, v)
	Expect(err).To(BeNil())

	value, err := dbClient.RequestCache.Get(k)
	Expect(err).To(BeNil())
	Expect(value).To(Equal(v))
}

func TestPayloadSetGet(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	key := []byte("keySetGetCache")
	resp := models.ResponseDetails{
		Status: 200,
		Body:   "body here",
	}

	payload := models.Payload{Response: resp}
	bts, err := json.Marshal(payload)
	Expect(err).To(BeNil())

	err = dbClient.RequestCache.Set(key, bts)
	Expect(err).To(BeNil())

	var p models.Payload
	payloadBts, err := dbClient.RequestCache.Get(key)
	err = json.Unmarshal(payloadBts, &p)
	Expect(err).To(BeNil())
	Expect(payload.Response.Body).To(Equal(p.Response.Body))

	defer dbClient.RequestCache.DeleteData()
}

func TestGetNonExistingBucket(t *testing.T) {
	RegisterTestingT(t)

	cache := cache.NewBoltDBCache(TestDB, []byte("somebucket"))

	_, err := cache.Get([]byte("whatever"))
	Expect(err).ToNot(BeNil())
	Expect(err).To(MatchError("Bucket \"somebucket\" not found!"))
}

func TestDeleteBucket(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	k := []byte("randomkeyhere")
	v := []byte("value")
	// checking whether bucket is okay
	err := dbClient.RequestCache.Set(k, v)
	Expect(err).To(BeNil())

	value, err := dbClient.RequestCache.Get(k)
	Expect(err).To(BeNil())
	Expect(value).To(Equal(v))

	// deleting bucket
	err = dbClient.RequestCache.DeleteData()
	Expect(err).To(BeNil())

	// deleting it again
	err = dbClient.RequestCache.DeleteData()
	Expect(err).ToNot(BeNil())
}

func TestGetAllRequestNoBucket(t *testing.T) {
	RegisterTestingT(t)

	cache := cache.NewBoltDBCache(TestDB, []byte("somebucket"))

	cache.CurrentBucket = []byte("no_bucket_for_TestGetAllRequestNoBucket")
	_, err := cache.GetAllValues()
	// expecting nil since this would mean that records were wiped
	Expect(err).To(BeNil())
}

func TestCorruptedPayloads(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	k := []byte("randomkeyhere")
	v := []byte("value")

	err := dbClient.RequestCache.Set(k, v)
	Expect(err).To(BeNil())

	// corrupted payloads should be just skipped
	payloads, err := dbClient.RequestCache.GetAllValues()
	Expect(err).To(BeNil())
	Expect(payloads).To(HaveLen(1))
}

func TestGetMultipleRecords(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()
	defer dbClient.RequestCache.DeleteData()

	// inserting some payloads
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://example.com/q=%d", i), nil)
		Expect(err).To(BeNil())
		dbClient.captureRequest(req)
	}

	// getting requests
	values, err := dbClient.RequestCache.GetAllValues()
	Expect(err).To(BeNil())

	for _, value := range values {
		if payload, err := models.NewPayloadFromBytes(value); err == nil {
			Expect(payload.Request.Method).To(Equal("GET"))
			Expect(payload.Response.Status).To(Equal(201))
		} else {
			t.Error(err)
		}
	}
}

func TestGetNonExistingKey(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()
	defer dbClient.RequestCache.DeleteData()

	// getting key
	_, err := dbClient.RequestCache.Get([]byte("should not be here"))
	Expect(err).ToNot(BeNil())
}

func TestSetGetEmptyValue(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()
	defer dbClient.RequestCache.DeleteData()

	err := dbClient.RequestCache.Set([]byte("shouldbe"), []byte(""))
	Expect(err).To(BeNil())
	// getting key
	_, err = dbClient.RequestCache.Get([]byte("shouldbe"))
	Expect(err).To(BeNil())
}

func TestGetAllKeys(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()
	defer dbClient.RequestCache.DeleteData()

	// inserting some payloads
	for i := 0; i < 5; i++ {
		dbClient.RequestCache.Set([]byte(fmt.Sprintf("key%d", i)), []byte("value"))
	}

	keys, err := dbClient.RequestCache.GetAllKeys()
	Expect(err).To(BeNil())
	Expect(keys).To(HaveLen(5))

	for k, v := range keys {
		Expect(k).To(HavePrefix("key"))
		Expect(v).To(BeTrue())
	}
}

func TestGetAllKeysEmpty(t *testing.T) {
	RegisterTestingT(t)

	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()
	defer dbClient.RequestCache.DeleteData()

	keys, err := dbClient.RequestCache.GetAllKeys()
	Expect(err).To(BeNil())
	Expect(keys).To(HaveLen(0))
}
