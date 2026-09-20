package client

import "testing"

func TestAuthorize(t *testing.T) {
	testCases := []struct {
		name          string
		request       AutherizaionRequest
		expectedError bool
	}{
		{
			name: "valid request",
			request: AutherizaionRequest{
				CardNumber: "4111111111111111",
				ExpiryDate: "12/25",
				Currency:   "USD",
				Amount:     100,
				Cvv:        "123",
			},
			expectedError: false,
		},
		{
			name: "invalid request - missing card number",
			request: AutherizaionRequest{
				CardNumber: "",
				ExpiryDate: "12/25",
				Currency:   "USD",
				Amount:     100,
				Cvv:        "123",
			},
			expectedError: true,
		},
		{
			name: "invalid request - missing expiry date",
			request: AutherizaionRequest{
				CardNumber: "4111111111111111",
				ExpiryDate: "",
				Currency:   "USD",
				Amount:     100,
				Cvv:        "123",
			},
			expectedError: true,
		},
		{
			name: "invalid request - missing currency",
			request: AutherizaionRequest{
				CardNumber: "4111111111111111",
				ExpiryDate: "12/25",
				Currency:   "",
				Amount:     100,
				Cvv:        "123",
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient("http://localhost:8080") //	 Use the actual base URL of your bank simulator
			_, err := client.Authorize(tc.request)
			if (err != nil) != tc.expectedError {
				t.Errorf("expected error: %v, got: %v", tc.expectedError, err)
			}
		})
	}
}
