# payment — docs

The **payment** plugin is togo's provider-agnostic payment subsystem. It defines the
`PaymentProvider` contract, a driver registry, and a `Service` stored on the kernel —
real gateways ship as separate **driver plugins** that register against it.

- **Marketplace:** https://to-go.dev/marketplace
- **Source:** https://github.com/togo-framework/payment

## Install

```bash
togo install togo-framework/payment            # the base
togo install togo-framework/payment-stripe     # + one or more gateway drivers
```

Select the active driver in `togo.yaml`/`.env`:

```env
PAYMENT_DRIVER=stripe   # or paymob | fawry | tap | moyasar | paytabs | payfort | lemonsqueezy | log
```

`log` is the safe default driver (no network — logs the intent), so the plugin is
usable before you wire a real gateway.

## The `PaymentProvider` contract

```go
type PaymentProvider interface {
    CreateCharge(ctx context.Context, req ChargeRequest) (*Charge, error)
    Refund(ctx context.Context, req RefundRequest) error
    CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutSession, error)
    CreateCustomer(ctx context.Context, c Customer) (*Customer, error)
    CreateSubscription(ctx context.Context, req SubscriptionRequest) (*Subscription, error)
    HandleWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error)
}
```

Key request types: `ChargeRequest{Amount Money, Customer, Description, Token, Metadata}`,
`CheckoutRequest{Amount Money, Customer, Items []LineItem, SuccessURL, CancelURL, Metadata}`,
`RefundRequest{ /* charge id; Amount nil = full refund */ }`.

## Using it from your app

The plugin registers via `togo.RegisterProviderFunc("payment", PriorityService, …)` and
stores a `*Service` on the kernel (`k.Get("payment")`). Fetch it with `FromKernel`:

```go
import "github.com/togo-framework/payment"

svc, ok := payment.FromKernel(k)
if !ok { /* payment plugin not installed */ }

charge, err := svc.CreateCharge(ctx, payment.ChargeRequest{
    Amount:   payment.Money{Value: 1000, Currency: "USD"},
    Customer: payment.Customer{Email: "buyer@example.com"},
    Token:    "<gateway-token>",
})
```

`Service` exposes the same methods as the interface (`CreateCharge`, `Refund`,
`CreateCheckoutSession`, `CreateCustomer`, `CreateSubscription`, `HandleWebhook`) and
forwards to the selected driver. Wire your own HTTP routes (checkout + webhook) and call
the service from them — the base does not mount REST routes for you, so you control the
surface and auth.

## Webhooks

In your webhook route, pass the raw body + headers straight through:

```go
ev, err := svc.HandleWebhook(ctx, headers, rawBody)
```

Every gateway driver **verifies the webhook signature** (HMAC / signature / secret-token,
depending on the gateway) and rejects forgeries when its webhook secret is configured —
see each driver's docs for the exact scheme and env var.

## Writing a driver

```go
func init() {
    payment.RegisterDriver("mygateway", func(k *togo.Kernel) (payment.PaymentProvider, error) {
        // read env, return a struct implementing PaymentProvider
    })
}
```

## Available drivers

| Driver | Gateway | Region |
|---|---|---|
| [`payment-stripe`](https://github.com/togo-framework/payment-stripe) | Stripe | Global |
| [`payment-paymob`](https://github.com/togo-framework/payment-paymob) | Paymob | Egypt / MENA |
| [`payment-fawry`](https://github.com/togo-framework/payment-fawry) | Fawry | Egypt |
| [`payment-tap`](https://github.com/togo-framework/payment-tap) | Tap | MENA |
| [`payment-moyasar`](https://github.com/togo-framework/payment-moyasar) | Moyasar | Saudi Arabia |
| [`payment-paytabs`](https://github.com/togo-framework/payment-paytabs) | PayTabs | MENA |
| [`payment-payfort`](https://github.com/togo-framework/payment-payfort) | Amazon PayFort | MENA |
| [`payment-lemonsqueezy`](https://github.com/togo-framework/payment-lemonsqueezy) | Lemon Squeezy | Global (MoR) |

Related: [`subscriptions`](https://github.com/togo-framework/subscriptions) (plans over payment) ·
[`billing`](https://github.com/togo-framework/billing) (API keys + usage metering).
