// Package payment is togo's payment subsystem: a PaymentProvider contract with a
// safe dev "log" driver. Real gateways (Stripe, Paymob, Fawry, Tap, Moyasar,
// PayTabs, PayFort, Lemon Squeezy, …) ship as driver plugins that call
// payment.RegisterDriver and depend on this package. Select one with PAYMENT_DRIVER.
//
// Install: `togo install togo-framework/payment` (blank-import registers it),
// then a driver, e.g. `togo install togo-framework/payment-stripe`.
package payment

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/togo-framework/togo"
)

// Money is an amount in the smallest currency unit (cents, piasters, halalas…).
type Money struct {
	Amount   int64  // minor units
	Currency string // ISO 4217, e.g. "USD", "EGP", "SAR"
}

// Customer identifies a payer.
type Customer struct {
	ID    string
	Email string
	Name  string
	Phone string
}

// LineItem is one item in a checkout.
type LineItem struct {
	Name     string
	Amount   Money
	Quantity int64
}

// ChargeRequest requests a one-off charge. Token is a payment-method / source
// token obtained from the provider's client SDK.
type ChargeRequest struct {
	Amount      Money
	Customer    Customer
	Description string
	Token       string
	Metadata    map[string]string
}

// Charge is the result of a charge.
type Charge struct {
	ID       string
	Status   string // succeeded | pending | failed
	Amount   Money
	Provider string
	Raw      map[string]any
}

// RefundRequest refunds a charge (full when Amount is nil, else partial).
type RefundRequest struct {
	ChargeID string
	Amount   *Money
}

// CheckoutRequest creates a hosted/redirect checkout session.
type CheckoutRequest struct {
	Amount     Money
	Customer   Customer
	Items      []LineItem
	SuccessURL string
	CancelURL  string
	Metadata   map[string]string
}

// CheckoutSession is a hosted checkout to redirect the customer to.
type CheckoutSession struct {
	ID  string
	URL string
}

// SubscriptionRequest starts a recurring subscription on a provider plan.
type SubscriptionRequest struct {
	Customer Customer
	PlanID   string
	Metadata map[string]string
}

// Subscription is a recurring subscription.
type Subscription struct {
	ID       string
	Status   string
	PlanID   string
	Provider string
}

// WebhookEvent is a provider webhook normalized to a common shape.
type WebhookEvent struct {
	Type     string // e.g. charge.succeeded, subscription.canceled
	ID       string
	Provider string
	Raw      map[string]any
}

// PaymentProvider is implemented by driver plugins. Not every gateway supports
// every operation — return a clear error for the unsupported ones.
type PaymentProvider interface {
	CreateCharge(ctx context.Context, req ChargeRequest) (*Charge, error)
	Refund(ctx context.Context, req RefundRequest) error
	CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutSession, error)
	CreateCustomer(ctx context.Context, c Customer) (string, error)
	CreateSubscription(ctx context.Context, req SubscriptionRequest) (*Subscription, error)
	HandleWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error)
}

// DriverFactory builds a PaymentProvider from the kernel (env-configured).
type DriverFactory func(k *togo.Kernel) (PaymentProvider, error)

var (
	regMu   sync.RWMutex
	drivers = map[string]DriverFactory{}
)

// RegisterDriver registers a payment driver by name (call from a plugin's init()).
func RegisterDriver(name string, f DriverFactory) {
	regMu.Lock()
	drivers[name] = f
	regMu.Unlock()
}

func init() {
	RegisterDriver("log", func(k *togo.Kernel) (PaymentProvider, error) { return &logProvider{k: k}, nil })

	togo.RegisterProviderFunc("payment", togo.PriorityService, func(k *togo.Kernel) error {
		name := os.Getenv("PAYMENT_DRIVER")
		if name == "" {
			name = "log" // safe dev default: don't charge, just log
		}
		regMu.RLock()
		f, ok := drivers[name]
		regMu.RUnlock()
		if !ok {
			return fmt.Errorf("payment: unknown driver %q (install its plugin, e.g. togo install togo-framework/payment-%s)", name, name)
		}
		p, err := f(k)
		if err != nil {
			return err
		}
		k.Set("payment", &Service{provider: p, driver: name})
		return nil
	})
}

// Service is the payment runtime stored on the kernel (k.Get("payment")).
type Service struct {
	provider PaymentProvider
	driver   string
}

// Provider returns the active driver implementation.
func (s *Service) Provider() PaymentProvider { return s.provider }

// Driver returns the active driver name.
func (s *Service) Driver() string { return s.driver }

func (s *Service) CreateCharge(ctx context.Context, req ChargeRequest) (*Charge, error) {
	return s.provider.CreateCharge(ctx, req)
}
func (s *Service) Refund(ctx context.Context, req RefundRequest) error {
	return s.provider.Refund(ctx, req)
}
func (s *Service) CreateCheckoutSession(ctx context.Context, req CheckoutRequest) (*CheckoutSession, error) {
	return s.provider.CreateCheckoutSession(ctx, req)
}
func (s *Service) CreateCustomer(ctx context.Context, c Customer) (string, error) {
	return s.provider.CreateCustomer(ctx, c)
}
func (s *Service) CreateSubscription(ctx context.Context, req SubscriptionRequest) (*Subscription, error) {
	return s.provider.CreateSubscription(ctx, req)
}
func (s *Service) HandleWebhook(ctx context.Context, headers map[string]string, body []byte) (*WebhookEvent, error) {
	return s.provider.HandleWebhook(ctx, headers, body)
}

// FromKernel fetches the payment service from the kernel container.
func FromKernel(k *togo.Kernel) (*Service, bool) {
	v, ok := k.Get("payment")
	if !ok {
		return nil, false
	}
	s, ok := v.(*Service)
	return s, ok
}

// logProvider logs instead of charging — the safe default for dev/tests.
type logProvider struct{ k *togo.Kernel }

func (l *logProvider) CreateCharge(_ context.Context, req ChargeRequest) (*Charge, error) {
	if l.k != nil && l.k.Log != nil {
		l.k.Log.Info("payment (log driver) charge", "amount", req.Amount.Amount, "currency", req.Amount.Currency)
	}
	return &Charge{ID: "log_charge", Status: "succeeded", Amount: req.Amount, Provider: "log"}, nil
}
func (l *logProvider) Refund(context.Context, RefundRequest) error { return nil }
func (l *logProvider) CreateCheckoutSession(_ context.Context, req CheckoutRequest) (*CheckoutSession, error) {
	return &CheckoutSession{ID: "log_cs", URL: req.SuccessURL}, nil
}
func (l *logProvider) CreateCustomer(context.Context, Customer) (string, error) {
	return "log_customer", nil
}
func (l *logProvider) CreateSubscription(_ context.Context, req SubscriptionRequest) (*Subscription, error) {
	return &Subscription{ID: "log_sub", Status: "active", PlanID: req.PlanID, Provider: "log"}, nil
}
func (l *logProvider) HandleWebhook(context.Context, map[string]string, []byte) (*WebhookEvent, error) {
	return &WebhookEvent{Type: "unknown", Provider: "log", Raw: map[string]any{}}, nil
}
