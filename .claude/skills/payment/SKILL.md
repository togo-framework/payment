---
name: payment
description: Accept payments in this togo app using the payment plugin and a provider (Stripe, Paymob, Fawry, Tap, Moyasar, PayTabs, PayFort, Lemon Squeezy).
---

Use the togo `payment` plugin:

1. Install a provider: `togo install togo-framework/payment-stripe` (or payment-paymob / payment-fawry / payment-tap…). Set its key in `.env` and `PAYMENT_DRIVER=stripe`.
2. Create a charge or hosted checkout via the kernel payment service; handle webhooks.
3. For recurring billing: `togo install togo-framework/subscriptions` and `togo-framework/billing` (API keys + token/usage metering).

Endpoints: `POST /api/payments`, checkout + webhook routes per the provider.
