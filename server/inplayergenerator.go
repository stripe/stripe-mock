package server

import "fmt"

func PostPaymentIntents() []byte {
	return []byte(fmt.Sprintf(`{
		"id": "pi_3JmzufH4sM2j3OdT05U65vh7",
		"object": "payment_intent",
		"amount": 1500,
		"amount_capturable": 0,
		"amount_received": 1500,
		"application": null,
		"application_fee_amount": null,
		"canceled_at": null,
		"cancellation_reason": null,
		"capture_method": "automatic",
		"charges": {
		  "object": "list",
		  "data": [
			{
			  "id": "%v",
			  "object": "charge",
			  "amount": 1500,
			  "amount_captured": 1500,
			  "amount_refunded": 0,
			  "application": null,
			  "application_fee": null,
			  "application_fee_amount": null,
			  "balance_transaction": "txn_3JmzufH4sM2j3OdT03JVmmoM",
			  "billing_details": {
				"address": {
				  "city": null,
				  "country": null,
				  "line1": null,
				  "line2": null,
				  "postal_code": null,
				  "state": null
				},
				"email": null,
				"name": "John Smith",
				"phone": null
			  },
			  "calculated_statement_descriptor": "INPLAYER",
			  "captured": true,
			  "created": 1634817506,
			  "currency": "eur",
			  "customer": "cus_KRtNIl3OeOWKiJ",
			  "description": "Vimeo playlist test",
			  "destination": null,
			  "dispute": null,
			  "disputed": false,
			  "failure_code": null,
			  "failure_message": null,
			  "fraud_details": {},
			  "invoice": null,
			  "livemode": false,
			  "metadata": {
				"customer": "73438bd2-fc3f-4131-b7b3-70a8a621545a",
				"merchant": "96586229-330d-4f74-9f36-a3765fbc1f52"
			  },
			  "on_behalf_of": null,
			  "order": null,
			  "outcome": {
				"network_status": "approved_by_network",
				"reason": null,
				"risk_level": "normal",
				"risk_score": 32,
				"seller_message": "Payment complete.",
				"type": "authorized"
			  },
			  "paid": true,
			  "payment_intent": "pi_3JmzufH4sM2j3OdT05U65vh7",
			  "payment_method": "card_1JmzayH4sM2j3OdTEQNqtubO",
			  "payment_method_details": {
				"card": {
				  "brand": "visa",
				  "checks": {
					"address_line1_check": null,
					"address_postal_code_check": null,
					"cvc_check": null
				  },
				  "country": "US",
				  "exp_month": 12,
				  "exp_year": 2021,
				  "fingerprint": "pdvCFaAciKUM7xYR",
				  "funding": "credit",
				  "installments": null,
				  "last4": "4242",
				  "network": "visa",
				  "three_d_secure": null,
				  "wallet": null
				},
				"type": "card"
			  },
			  "receipt_email": null,
			  "receipt_number": null,
			  "receipt_url": "https://pay.stripe.com/receipts/acct_17TYuaH4sM2j3OdT/ch_3JmzufH4sM2j3OdT0Snk3DkV/rcpt_KRti48RSalsv90cEkZGac5o57LS28AY",
			  "refunded": false,
			  "refunds": {
				"object": "list",
				"data": [],
				"has_more": false,
				"total_count": 0,
				"url": "/v1/charges/ch_3JmzufH4sM2j3OdT0Snk3DkV/refunds"
			  },
			  "review": null,
			  "shipping": null,
			  "source": {
				"id": "card_1JmzayH4sM2j3OdTEQNqtubO",
				"object": "card",
				"address_city": null,
				"address_country": null,
				"address_line1": null,
				"address_line1_check": null,
				"address_line2": null,
				"address_state": null,
				"address_zip": null,
				"address_zip_check": null,
				"brand": "Visa",
				"country": "US",
				"customer": "cus_KRtNIl3OeOWKiJ",
				"cvc_check": null,
				"dynamic_last4": null,
				"exp_month": 12,
				"exp_year": 2021,
				"fingerprint": "pdvCFaAciKUM7xYR",
				"funding": "credit",
				"last4": "4242",
				"metadata": {
				  "customer_uuid": "73438bd2-fc3f-4131-b7b3-70a8a621545a",
				  "merchant_uuid": "96586229-330d-4f74-9f36-a3765fbc1f52"
				},
				"name": "John Smith",
				"tokenization_method": null
			  },
			  "source_transfer": null,
			  "statement_descriptor": null,
			  "statement_descriptor_suffix": null,
			  "status": "succeeded",
			  "transfer_data": null,
			  "transfer_group": null
			}
		  ],
		  "has_more": false,
		  "total_count": 1,
		  "url": "/v1/charges?payment_intent=pi_3JmzufH4sM2j3OdT05U65vh7"
		},
		"client_secret": "pi_3JmzufH4sM2j3OdT05U65vh7_secret_JicpL04KQO3GL4P3KvEomvkC4",
		"confirmation_method": "manual",
		"created": 1634817505,
		"currency": "eur",
		"customer": "cus_KRtNIl3OeOWKiJ",
		"description": "Vimeo playlist test",
		"invoice": null,
		"last_payment_error": null,
		"livemode": false,
		"metadata": {
		  "customer": "73438bd2-fc3f-4131-b7b3-70a8a621545a",
		  "merchant": "96586229-330d-4f74-9f36-a3765fbc1f52"
		},
		"next_action": null,
		"on_behalf_of": null,
		"payment_method": null,
		"payment_method_options": {
		  "card": {
			"installments": null,
			"network": null,
			"request_three_d_secure": "automatic"
		  }
		},
		"payment_method_types": [
		  "card"
		],
		"receipt_email": null,
		"review": null,
		"setup_future_usage": null,
		"shipping": null,
		"source": "card_1JmzayH4sM2j3OdTEQNqtubO",
		"statement_descriptor": null,
		"statement_descriptor_suffix": null,
		"status": "succeeded",
		"transfer_data": null,
		"transfer_group": null
	  }`, randomID("ch")))
}
