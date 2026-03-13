package aws

import (
	"testing"

	ec2_t "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatEC2Tags_PreservesOrderAndEmptyValue(t *testing.T) {
	keyA := "env"
	valA := "prod"
	keyB := "owner"
	valB := ""
	keyEmpty := ""
	valIgnored := "ignored"

	tags := []ec2_t.Tag{
		{Key: &keyA, Value: &valA},
		{Key: &keyB, Value: &valB},
		{Key: &keyA, Value: &valA},
		{Key: &keyEmpty, Value: &valIgnored},
	}

	assert.Equal(t, []string{"env: prod", "owner: "}, formatEC2Tags(tags, true))
}

func TestFormatEC2Tags_DisabledReturnsNil(t *testing.T) {
	key := "env"
	val := "prod"

	assert.Nil(t, formatEC2Tags([]ec2_t.Tag{{Key: &key, Value: &val}}, false))
}

func TestFormatStringMapTags_DisabledReturnsNil(t *testing.T) {
	assert.Nil(t, formatStringMapTags(map[string]string{"env": "prod"}, false))
}
