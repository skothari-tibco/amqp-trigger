package amqp

import (
	"testing"

	"github.com/project-flogo/core/support"
	"github.com/project-flogo/core/trigger"
	"github.com/stretchr/testify/assert"
)

func TestTrigger_Register(t *testing.T) {

	ref := support.GetRef(&AMQPTrigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(t, f)
}
