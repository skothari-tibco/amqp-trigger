package amqp

import (
	"encoding/json"
	"testing"

	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/support"
	"github.com/project-flogo/core/support/test"
	"github.com/project-flogo/core/trigger"
	"github.com/stretchr/testify/assert"
)

const testConfig string = `{
	"id": "flogo-amqp",
	"ref": "github.com/skothari-tibco/amqp-trigger",
	"settings": {
	  "amqpURI": "amqp://zfdqtzvv:i6bIvbC1m81FRC0AqUZzNHG5EssTvrAL@spider.rmq.cloudamqp.com/zfdqtzvv",
	  "consumerTag": "client1",
	  "certPem":"/Users/skothari-tibco/Desktop/cert.pem",
	  "keyPem":"/Users/skothari-tibco/Desktop/key.pem"
	},
	"handlers": [
	  {
			"action":{
				"id":"dummy"
			},
			"settings": {
				"queue": "hello",
				"exchange": "test-excahnge",
				"bindingKey" : "red",
				"exchangeType":"direct"
			}
	  }
	]
	
  }`

func TestTrigger_Register(t *testing.T) {

	ref := support.GetRef(&AMQPTrigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(t, f)
}

func TestAMQPTrigger_Initialize(t *testing.T) {
	f := &Factory{}

	config := &trigger.Config{}
	err := json.Unmarshal([]byte(testConfig), config)
	assert.Nil(t, err)

	actions := map[string]action.Action{"dummy": test.NewDummyAction(func() {
		//do nothing
	})}

	trg, err := test.InitTrigger(f, config, actions)
	assert.Nil(t, err)
	assert.NotNil(t, trg)

	err = trg.Start()
	assert.Nil(t, err)

}
