package amqp

type Settings struct {
	AmqpURI     string `md:"amqURI"`
	ConsumerTag string `md:"consumerTag"`

	CertPem string `md:"certPem"`
	KeyPem  string `md:"keyPem"`
}

type HandlerSettings struct {
	Queue        string `md:"queue"`
	Exchange     string `md:"exchange"`
	BindingKey   string `md:"bindingKey"`
	ExchangeType string `md:"exchangeType"`
}

type Output struct {
	Data interface{} `md:"data"`
}

func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"data": o.Data,
	}
}

func (o *Output) FromMap(values map[string]interface{}) error {

	o.Data = values["data"]

	return nil
}
