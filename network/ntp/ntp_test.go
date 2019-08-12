package ntp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTimeProof(t *testing.T) {
	assert.NoError(t, TimeProof())
}
