package resource

import (
	"testing"

	"kubecloud/common/utils"
	_ "kubecloud/test/mock"

	"github.com/stretchr/testify/assert"
)

var opaqueRequest = SecretRequest{
	Creator:     "admin",
	Name:        "my-secret-opaque",
	Description: "this is my secret",
	Type:        "Opaque",
	Data: map[string]string{
		"field1": "data1",
		"field2": "data2",
		"field3": "data3",
	},
}

var basicAuthRequest = SecretRequest{
	Creator:     "admin",
	Name:        "my-secret-tls",
	Description: "this is my secret",
	Type:        "kubernetes.io/basic-auth",
	Data: map[string]string{
		"username": "username1",
		"password": "password1",
	},
}

var tlsRequest = SecretRequest{
	Creator:     "admin",
	Name:        "my-secret-basic-auth",
	Description: "this is my secret",
	Type:        "kubernetes.io/tls",
	Data: map[string]string{
		"tls.crt": `-----BEGIN CERTIFICATE-----
MIIDVjCCAj4CCQCGx9KHhx2XxTANBgkqhkiG9w0BAQUFADBtMQ4wDAYDVQQDDAVh
LmNvbTELMAkGA1UECgwCZmYxCzAJBgNVBAsMAmZmMQowCAYDVQQHDAFhMRAwDgYD
VQQIDAdBbGFiYW1hMQswCQYDVQQGEwJVUzEWMBQGCSqGSIb3DQEJARYHYUBhLmNv
bTAeFw0xOTA0MTYwMjUyMzRaFw0yMDA0MTUwMjUyMzRaMG0xDjAMBgNVBAMMBWEu
Y29tMQswCQYDVQQKDAJmZjELMAkGA1UECwwCZmYxCjAIBgNVBAcMAWExEDAOBgNV
BAgMB0FsYWJhbWExCzAJBgNVBAYTAlVTMRYwFAYJKoZIhvcNAQkBFgdhQGEuY29t
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxyUNKqS6Isd+SW9Vl7tQ
2jIVWWm5kXLiWFS7lQQO0LTTfPEvddhUkv/yAmoEvcPtUA/trtNAnfJNWPQajlIh
H9IsHpYBM7cAyOfnApNaRHor/qaFr6S66ASX6hHVchqVmfQpGwquVhzV4EZb6wy1
c+omwZ+b97WS89SDJHUburFDNzxeWZxGQtRrxam/tmtLcLYuXNeIBZ0Ci4om2jhN
wp3NqGIuiwejfh0Ngk8TlyPLadtzzKe1qce7WM+J8auPU0xy8xlNuCFyWRbfkFDf
THzj40rnskicjQeq+nVY0VuYThZX0RID5c/7GEWKK8FApZV9Dw7sb/3Uuii4S4Cm
fwIDAQABMA0GCSqGSIb3DQEBBQUAA4IBAQBlY+J0mJwa9EZsld1kGp1lyTyaltOB
m+dA7WQNuFrYiH5KmI/erM7cmDbJGBrH6dpN6t7dytg3DcdT2Nq8cTzWVqaKhMqX
+pyY0Uj0yWq+4Q4aD5WBkg8fC2rzMaY24Q6ifRTq8MJbGZ5feCoIysrzdG9m0cFa
rVLqXTm8NcvXV9qLFWOB8xa1i3oJxevzv++1FL7T/sh908nJB9ky7x4w7Ao5B8bG
vrlKFqlLXa8rT1bA7vnWfXOl+d2hhiLBLYAzQoeAyGxJwODR/J9Z1tBWzcZjUhs7
9QZewHQb8Ni5/CTSqxU0M0u8r28cmeLDrkT0VSoY7nXkh6l4ncZZ6yxW
-----END CERTIFICATE-----
`,
		"tls.key": `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAxyUNKqS6Isd+SW9Vl7tQ2jIVWWm5kXLiWFS7lQQO0LTTfPEv
ddhUkv/yAmoEvcPtUA/trtNAnfJNWPQajlIhH9IsHpYBM7cAyOfnApNaRHor/qaF
r6S66ASX6hHVchqVmfQpGwquVhzV4EZb6wy1c+omwZ+b97WS89SDJHUburFDNzxe
WZxGQtRrxam/tmtLcLYuXNeIBZ0Ci4om2jhNwp3NqGIuiwejfh0Ngk8TlyPLadtz
zKe1qce7WM+J8auPU0xy8xlNuCFyWRbfkFDfTHzj40rnskicjQeq+nVY0VuYThZX
0RID5c/7GEWKK8FApZV9Dw7sb/3Uuii4S4CmfwIDAQABAoIBAFBxRg1ItyufXAL7
5x5Aext4iakxjNUVDBtUNuWt2vIxaRCaOGqo9RjgLHkPLFUfVLg3hvJMwVhL7TSW
fjepM1owQsQkId4q+TUuf+FJdngcxbK6v1vA9gMM//R8ObU6yd2DhCs5OEzunvJW
WYDsBEwLdJZYC9+WyAKYgFT0vTu+8aGsINW89rRqXi1k6kZY1CMM1/xI13QNOgCP
F20NpAm9bPYxO003PUYl2yEvSkOUrqHggiRngCl3BsT77c9HcB6Hn0Ib9LVXXUOG
Y3bZSYpCYkv0kCruOjNoNyxJ8j4WY9NTfufwq4vm/K41MweSOnfN2RwAMX3JL3VE
A7U4YwECgYEA+iWPExeQ5Dx9lLAQJlRc/s9mEv+68iLWyRSdSJo67a7n2iXs+zRy
3X98+Z2MTF1MpZ4uiRzi7G8x1ppIKuUHZ/3ReJIvwIeLbZ+Xq1jRPJUlD3rmv8C/
c4aXKRaq6+1+n6pKS1Ct18GsN3d0uFyOoR74o5R9uhjf4pfS/OB48P8CgYEAy836
XF2snEFWKs1GhAdUKX1b89wN8AhiixBkEr+Mkjke0Wz879D7T9IXtxGUDB4vIBWz
H/Xy8j2ZzsX8ygnuYOQ0r+ueuZyo6IIIJOqLmAttLRwJVQCC2p9ZzkUjyGYuyECz
/yl9MbwVNuovVYegOaPVRDhhldjgEE0GPjmayoECgYEAjvj8p7rmc60nUd63vFCn
vnQoMV+9KDxFazS/GQod+E/p8MOQiZvWs0b01W75C4SgFGEu0+uQv/ZmE/Smnu28
p/Fo0nMrm+1dAdEfzS28mdXdEtX6IUs3of4hU7jDBIn/v56DTDzWv+TQW/uI2P79
/pVHI6fxnHYvMMH6M9LRDV0CgYA40+x1iOEyiL1gHfEFq416LCxRqRBx18SyhhWB
bMvjke3X843ryNfqf+iA8XPYlSoKxkI2LTxa83ZJw8cjBvXjKn7OduLBWr92ZZuj
v4rBEJ6Wr3SisQvLrhc6fujlXii5SeFmysjP72Py9gXQ0YqJx/cVmKsNP3Xq1a9h
9moFAQKBgHe3gCatVnooxRIkUTiNm4YduzU5nguUX8OaaxcqNBkfiy+fVeTmUku+
8fQp0daSjpKTeaELita6ItEDoKddUZ4qVz62QRayzGMYLzn1wuvwcSTt0WkANFvC
cC9LFPBmahSK8wLMlfGiW8WQ1mQpi27l0xaf62jUVBsXxoqG9xiX
-----END RSA PRIVATE KEY-----
`,
	},
}

func TestSecret(t *testing.T) {
	res, err := NewSecretRes("cluster1", nil)
	assert.NotNil(t, res)
	assert.Nil(t, err)

	t.Run("GetTypes", func(t *testing.T) {
		assert.NotEmpty(t, res.GetTypes())
	})

	t.Run("Validate", func(t *testing.T) {
		t.Run("Positive", func(t *testing.T) {
			t.Parallel()
			positiveCases := map[string]*SecretRequest{
				"Opaque":    &opaqueRequest,
				"BasicAuth": &basicAuthRequest,
				"TLS":       &tlsRequest,
			}
			for name, request := range positiveCases {
				request := request
				t.Run(name, func(t *testing.T) {
					assert.Nil(t, res.Validate(request))
				})
			}
		})
		t.Run("Negative", func(t *testing.T) {
			t.Parallel()
			t.Run("InvalidType", func(t *testing.T) {
				var request SecretRequest
				if err := utils.DeepCopy(&request, &tlsRequest); err != nil {
					assert.Nil(t, err)
				}
				request.Type = ""
				assert.NotNil(t, res.Validate(&request))
			})
			t.Run("TLS", func(t *testing.T) {
				t.Run("NoCrt", func(t *testing.T) {
					var request SecretRequest
					if err := utils.DeepCopy(&request, &tlsRequest); err != nil {
						assert.Nil(t, err)
					}
					delete(request.Data, "tls.crt")
					assert.NotNil(t, res.Validate(&request))
				})
				t.Run("NoKey", func(t *testing.T) {
					var request SecretRequest
					if err := utils.DeepCopy(&request, &tlsRequest); err != nil {
						assert.Nil(t, err)
					}
					delete(request.Data, "tls.key")
					assert.NotNil(t, res.Validate(&request))
				})
				t.Run("Invalid", func(t *testing.T) {
					var request SecretRequest
					if err := utils.DeepCopy(&request, &tlsRequest); err != nil {
						assert.Nil(t, err)
					}
					key := request.Data["tls.key"]
					request.Data["tls.key"] = request.Data["tls.crt"]
					request.Data["tls.crt"] = key
					err := res.Validate(&request)
					assert.NotNil(t, err)
				})
			})
			t.Run("BasicAuth", func(t *testing.T) {
				t.Run("NoUsername", func(t *testing.T) {
					var request SecretRequest
					if err := utils.DeepCopy(&request, &basicAuthRequest); err != nil {
						assert.Nil(t, err)
					}
					delete(request.Data, "username")
					assert.NotNil(t, res.Validate(&request))
				})
				t.Run("NoPassword", func(t *testing.T) {
					var request SecretRequest
					if err := utils.DeepCopy(&request, &basicAuthRequest); err != nil {
						assert.Nil(t, err)
					}
					delete(request.Data, "password")
					assert.NotNil(t, res.Validate(&request))
				})
			})
		})
	})
}
