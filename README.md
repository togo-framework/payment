# payment

togo's **payment subsystem** — a provider-agnostic `PaymentProvider` contract with
a safe dev `log` driver. Real gateways ship as **driver plugins** that call
`payment.RegisterDriver`; pick one with `PAYMENT_DRIVER`.

```bash
togo install togo-framework/payment            # the base
togo install togo-framework/payment-stripe     # a gateway driver
```

Drivers available: `payment-stripe`, `payment-paymob`, `payment-fawry`,
`payment-tap`, `payment-moyasar`, `payment-paytabs`, `payment-payfort`,
`payment-lemonsqueezy`.

## Configure

```env
PAYMENT_DRIVER=stripe        # or paymob | fawry | tap | moyasar | … | log (default)
# + the selected driver's env (e.g. STRIPE_SECRET_KEY)
```

## Use

```go
import "github.com/togo-framework/payment"

svc, _ := payment.FromKernel(k)
ch, err := svc.CreateCharge(ctx, payment.ChargeRequest{
    Amount:   payment.Money{Amount: 5000, Currency: "USD"}, // 50.00
    Customer: payment.Customer{Email: "a@b.com"},
    Token:    "tok_visa",
})

cs, _ := svc.CreateCheckoutSession(ctx, payment.CheckoutRequest{
    Amount: payment.Money{Amount: 5000, Currency: "EGP"},
    SuccessURL: "https://app/success", CancelURL: "https://app/cancel",
})
// redirect the customer to cs.URL
```

`Money` is in the smallest currency unit (cents/piasters/halalas). The
`PaymentProvider` interface covers charges, refunds, hosted checkout, customers,
subscriptions, and normalized webhooks — drivers return a clear error for
operations a gateway doesn't support.

MIT
