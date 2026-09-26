package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"workshop/internal/domain"
	"workshop/internal/usecase"
)

// Isolated mock stores for the 5-Gates security certification suite

type fgUserStore struct {
	user map[string]domain.User
}

func (s *fgUserStore) FindByUsername(_ context.Context, username string) (domain.User, error) {
	u, ok := s.user[username]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (s *fgUserStore) FindByID(_ context.Context, id string) (domain.User, error) {
	for _, u := range s.user {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (s *fgUserStore) UpdatePassword(_ context.Context, userID, newPasswordHash string) error {
	for k, u := range s.user {
		if u.ID == userID {
			u.PasswordHash = newPasswordHash
			u.RequiresPasswordChange = false
			s.user[k] = u
			return nil
		}
	}
	return domain.ErrNotFound
}

type fgTechStore struct {
	mu     sync.Mutex
	techs  map[string]domain.Technician
	byUser map[string]string
}

func newFgTechStore() *fgTechStore {
	return &fgTechStore{
		techs:  make(map[string]domain.Technician),
		byUser: make(map[string]string),
	}
}

func (s *fgTechStore) add(id, userID, specialty string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.techs[id] = domain.Technician{ID: id, UserID: userID, Specialty: specialty}
	s.byUser[userID] = id
}

func (s *fgTechStore) ListWorkload(_ context.Context) ([]domain.TechnicianWorkload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []domain.TechnicianWorkload
	for _, t := range s.techs {
		list = append(list, domain.TechnicianWorkload{Technician: t})
	}
	return list, nil
}

func (s *fgTechStore) FindByID(_ context.Context, id string) (domain.Technician, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.techs[id]
	if !ok {
		return domain.Technician{}, domain.ErrNotFound
	}
	return t, nil
}

func (s *fgTechStore) FindByUserID(_ context.Context, userID string) (domain.Technician, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byUser[userID]
	if !ok {
		return domain.Technician{}, domain.ErrNotFound
	}
	return s.techs[id], nil
}

type fgOrderStore struct {
	mu          sync.Mutex
	orders      map[string]domain.ServiceOrder
	transitions map[string][]domain.StatusTransition
	seq         int
}

func newFgOrderStore(initial ...domain.ServiceOrder) *fgOrderStore {
	m := make(map[string]domain.ServiceOrder)
	for _, o := range initial {
		m[o.ID] = o
	}
	return &fgOrderStore{
		orders:      m,
		transitions: make(map[string][]domain.StatusTransition),
	}
}

func (s *fgOrderStore) Save(_ context.Context, order domain.ServiceOrder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.ID] = order
	return nil
}

func (s *fgOrderStore) FindByID(_ context.Context, id string) (domain.ServiceOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return domain.ServiceOrder{}, domain.ErrNotFound
	}
	return o, nil
}

func (s *fgOrderStore) List(_ context.Context, status string) ([]usecase.ServiceOrderSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []usecase.ServiceOrderSummary
	for _, o := range s.orders {
		if status == "" || string(o.Status) == status {
			list = append(list, usecase.ServiceOrderSummary{Order: o})
		}
	}
	return list, nil
}

func (s *fgOrderStore) ListByVehicle(_ context.Context, _ string) ([]domain.ServiceOrder, error) {
	return nil, nil
}

func (s *fgOrderStore) UpdateStatus(_ context.Context, order domain.ServiceOrder, transition domain.StatusTransition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.ID] = order
	s.transitions[order.ID] = append(s.transitions[order.ID], transition)
	return nil
}

func (s *fgOrderStore) ListTransition(_ context.Context, orderID string) ([]domain.StatusTransition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transitions[orderID], nil
}

func (s *fgOrderStore) CountByStatus(_ context.Context) (map[string]int, error) {
	return map[string]int{}, nil
}

func (s *fgOrderStore) NextOrderNumber(_ context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return fmt.Sprintf("OS-%04d", s.seq), nil
}

type fgAssignmentStore struct {
	mu   sync.Mutex
	asgs map[string]domain.Assignment
}

func newFgAssignmentStore() *fgAssignmentStore {
	return &fgAssignmentStore{asgs: make(map[string]domain.Assignment)}
}

func (s *fgAssignmentStore) set(a domain.Assignment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asgs[a.ID] = a
}

func (s *fgAssignmentStore) Save(_ context.Context, a domain.Assignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.IsActive {
		for _, ex := range s.asgs {
			if ex.IsActive && ex.ServiceOrderID == a.ServiceOrderID && ex.ID != a.ID {
				return fmt.Errorf("%w: order already has active technician", domain.ErrConflict)
			}
		}
	}
	s.asgs[a.ID] = a
	return nil
}

func (s *fgAssignmentStore) FindActiveByServiceOrder(_ context.Context, orderID string) (domain.Assignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.asgs {
		if a.ServiceOrderID == orderID && a.IsActive {
			return a, nil
		}
	}
	return domain.Assignment{}, domain.ErrNotFound
}

func (s *fgAssignmentStore) FindActiveByTechnician(_ context.Context, techID string) (domain.Assignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.asgs {
		if a.TechnicianID == techID && a.IsActive {
			return a, nil
		}
	}
	return domain.Assignment{}, domain.ErrNotFound
}

func (s *fgAssignmentStore) FindLatestByServiceOrder(_ context.Context, orderID string) (domain.Assignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest domain.Assignment
	found := false
	for _, a := range s.asgs {
		if a.ServiceOrderID == orderID {
			if !found || a.AssignedAt.After(latest.AssignedAt) {
				latest = a
				found = true
			}
		}
	}
	if !found {
		return domain.Assignment{}, domain.ErrNotFound
	}
	return latest, nil
}

func (s *fgAssignmentStore) ReleaseByServiceOrder(_ context.Context, orderID string, releasedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, a := range s.asgs {
		if a.ServiceOrderID == orderID && a.IsActive {
			a.Release(releasedAt)
			s.asgs[id] = a
		}
	}
	return nil
}

type fgDiagStore struct {
	mu    sync.Mutex
	diags map[string]domain.Diagnostic
}

func newFgDiagStore() *fgDiagStore {
	return &fgDiagStore{diags: make(map[string]domain.Diagnostic)}
}

func (s *fgDiagStore) Save(_ context.Context, d domain.Diagnostic) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diags[d.ServiceOrderID] = d
	return nil
}

func (s *fgDiagStore) FindByServiceOrder(_ context.Context, orderID string) (domain.Diagnostic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.diags[orderID]
	if !ok {
		return domain.Diagnostic{}, domain.ErrNotFound
	}
	return d, nil
}

func (s *fgDiagStore) ListByVehicle(_ context.Context, _ string) ([]domain.Diagnostic, error) {
	return nil, nil
}

type fgIntervStore struct {
	mu      sync.Mutex
	intervs map[string][]domain.Intervention
}

func newFgIntervStore() *fgIntervStore {
	return &fgIntervStore{intervs: make(map[string][]domain.Intervention)}
}

func (s *fgIntervStore) Save(_ context.Context, i domain.Intervention) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intervs[i.ServiceOrderID] = append(s.intervs[i.ServiceOrderID], i)
	return nil
}

func (s *fgIntervStore) ListByServiceOrder(_ context.Context, orderID string) ([]domain.Intervention, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.intervs[orderID]
	res := make([]domain.Intervention, len(list))
	copy(res, list)
	return res, nil
}

func (s *fgIntervStore) ListByVehicle(_ context.Context, _ string) ([]domain.Intervention, error) {
	return nil, nil
}

func (s *fgIntervStore) FindByID(_ context.Context, _ string) (domain.Intervention, error) {
	return domain.Intervention{}, domain.ErrNotFound
}

// asCallerWithID injects a specific caller identity into the request context.
func asCallerWithID(request *http.Request, role domain.Role, userID string) *http.Request {
	return request.WithContext(context.WithValue(
		request.Context(), callerContextKey, caller{UserID: userID, Role: role},
	))
}

// =========================================================================
// GATE 1: Authentication + Global RBAC + Rate Limiting Lifecycle
// =========================================================================
func TestGate1_AuthenticationAndGlobalAuthorization(t *testing.T) {
	custStore := &fakeCustomerStore{}
	custHandler := NewCustomerHandler(usecase.NewCustomerUseCase(custStore, sequentialIDTest(), testClock()))
	vehHandler := NewVehicleHandler(usecase.NewVehicleUseCase(newFakeVehicleStore("v1"), custStore, sequentialIDTest(), testClock()))
	warHandler := newWarrantyHandler(t, newFakeWarrantyStore())
	techHandler := NewTechnicianHandler(usecase.NewTechnicianUseCase(&fakeTechnicianStore{}, sequentialIDTest(), testClock()))

	// 1. Unauthenticated requests to protected endpoints return 401
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/customer", nil)
	unauthRec := httptest.NewRecorder()
	custHandler.List(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for anonymous request, got %d", unauthRec.Code)
	}

	// 2. Global catalogs: 403 for Technician, 200 for Administrator
	catalogs := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{"Customer", custHandler.List, "/api/customer"},
		{"Vehicle", vehHandler.List, "/api/vehicle"},
		{"Warranty", warHandler.List, "/api/warranty"},
		{"Technician", techHandler.List, "/api/technician"},
	}

	for _, c := range catalogs {
		// Technician attempt -> 403
		techReq := asCallerWithID(httptest.NewRequest(http.MethodGet, c.path, nil), domain.RoleTechnician, "user-tech")
		techRec := httptest.NewRecorder()
		c.handler(techRec, techReq)
		if techRec.Code != http.StatusForbidden {
			t.Errorf("expected 403 for technician on %s, got %d", c.name, techRec.Code)
		}

		// Administrator attempt -> 200
		adminReq := asCallerWithID(httptest.NewRequest(http.MethodGet, c.path, nil), domain.RoleAdministrator, "user-admin")
		adminRec := httptest.NewRecorder()
		c.handler(adminRec, adminReq)
		if adminRec.Code != http.StatusOK {
			t.Errorf("expected 200 for admin on %s, got %d", c.name, adminRec.Code)
		}
	}

	// 3. Rate Limiter Lifecycle:
	// - Malformed JSON does NOT consume quota (returns 400)
	// - 5 failed attempts return 401
	// - 6th attempt returns 429 Too Many Requests with Retry-After header
	limiter := NewLoginRateLimiter()
	t.Cleanup(func() { limiter.Close() })

	hashedPass, _ := bcrypt.GenerateFromPassword([]byte("ValidPassword!"), bcrypt.MinCost)
	userRepo := &fgUserStore{
		user: map[string]domain.User{
			"admin": {
				ID:           "u1",
				Username:     "admin",
				PasswordHash: string(hashedPass),
				Role:         domain.RoleAdministrator,
				FullName:     "Admin",
			},
		},
	}
	issuer := NewTokenIssuer("secret_test_token_2026", 10*time.Hour)
	authHandler := NewAuthHandler(usecase.NewAuthenticateUser(userRepo, issuer, testClock()), limiter)

	// Malformed JSON check (must return 400 without consuming failure quota)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/session", strings.NewReader("{invalid_json"))
		req.RemoteAddr = "192.168.1.100:1234"
		rec := httptest.NewRecorder()
		authHandler.SignIn(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for malformed json, got %d", rec.Code)
		}
	}

	// 5 failed password attempts
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/session", strings.NewReader(`{"username":"admin","password":"WrongPassword!"}`))
		req.RemoteAddr = "192.168.1.100:1234"
		rec := httptest.NewRecorder()
		authHandler.SignIn(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("attempt %d: expected 401, got %d", i, rec.Code)
		}
	}

	// 6th attempt: blocked with 429
	reqBlocked := httptest.NewRequest(http.MethodPost, "/api/session", strings.NewReader(`{"username":"admin","password":"WrongPassword!"}`))
	reqBlocked.RemoteAddr = "192.168.1.100:1234"
	recBlocked := httptest.NewRecorder()
	authHandler.SignIn(recBlocked, reqBlocked)
	if recBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests on 6th attempt, got %d", recBlocked.Code)
	}
	if retryAfter := recBlocked.Header().Get("Retry-After"); retryAfter == "" {
		t.Errorf("expected Retry-After header on 429 response")
	}
}

// =========================================================================
// GATE 2: Ownership + Assignment Integrity (Double Lock Enforcement)
// =========================================================================
func TestGate2_OwnershipAndAssignmentIntegrity(t *testing.T) {
	orders := newFgOrderStore(orderFixture(t, "order-1", domain.StatusReceived))
	assignments := newFgAssignmentStore()
	assignments.set(domain.Assignment{
		ID:             "asg-1",
		ServiceOrderID: "order-1",
		TechnicianID:   "tech-1",
		IsActive:       true,
	})

	techStore := newFgTechStore()
	techStore.add("tech-1", "usr-tech1", "Motor")
	techStore.add("tech-2", "usr-tech2", "Frenos")

	orderUseCase := usecase.NewServiceOrderUseCase(orders, newFakeVehicleStore("vehicle-1"), assignments, techStore, sequentialIDTest(), testClock())
	orderHandler := NewServiceOrderHandler(orderUseCase)

	diagStore := newFgDiagStore()
	diagUseCase := usecase.NewDiagnosticUseCase(diagStore, orders, assignments, techStore, sequentialIDTest(), testClock())
	diagHandler := NewDiagnosticHandler(diagUseCase)

	intervStore := newFgIntervStore()
	intervUseCase := usecase.NewInterventionUseCase(intervStore, orders, assignments, techStore, sequentialIDTest(), testClock())
	intervHandler := NewInterventionHandler(intervUseCase)

	// 1. Tech 2 (usr-tech2) attempts to advance order-1 (assigned to tech-1) -> MUST BE 403 Forbidden
	reqAdv := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-1/status", strings.NewReader(`{"status":"IN_DIAGNOSIS"}`)), domain.RoleTechnician, "usr-tech2")
	reqAdv.SetPathValue("serviceOrderId", "order-1")
	recAdv := httptest.NewRecorder()
	orderHandler.Advance(recAdv, reqAdv)
	if recAdv.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-assigned tech advancing status, got %d: %s", recAdv.Code, recAdv.Body.String())
	}

	// 2. Tech 2 attempts to record diagnostic on order-1 -> MUST BE 403 Forbidden
	reqDiag := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-1/diagnostic", strings.NewReader(`{"finding":"falla","componentToRepair":"motor"}`)), domain.RoleTechnician, "usr-tech2")
	reqDiag.SetPathValue("serviceOrderId", "order-1")
	recDiag := httptest.NewRecorder()
	diagHandler.Record(recDiag, reqDiag)
	if recDiag.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-assigned tech recording diagnostic, got %d", recDiag.Code)
	}

	// 3. Tech 2 attempts client-side ID spoofing: sending "technicianId": "tech-1" in body
	reqSpoof := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-1/diagnostic", strings.NewReader(`{"finding":"falla","componentToRepair":"motor","technicianId":"tech-1"}`)), domain.RoleTechnician, "usr-tech2")
	reqSpoof.SetPathValue("serviceOrderId", "order-1")
	recSpoof := httptest.NewRecorder()
	diagHandler.Record(recSpoof, reqSpoof)
	if recSpoof.Code != http.StatusForbidden && recSpoof.Code != http.StatusBadRequest {
		t.Errorf("expected 400 or 403 when spoofing technicianId in body, got %d", recSpoof.Code)
	}

	// 4. Assigned Tech 1 (usr-tech1) records diagnostic -> MUST BE 201 Created
	reqLegitDiag := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-1/diagnostic", strings.NewReader(`{"finding":"falla real","componentToRepair":"filtro"}`)), domain.RoleTechnician, "usr-tech1")
	reqLegitDiag.SetPathValue("serviceOrderId", "order-1")
	recLegitDiag := httptest.NewRecorder()
	diagHandler.Record(recLegitDiag, reqLegitDiag)
	if recLegitDiag.Code != http.StatusCreated {
		t.Fatalf("expected 201 for assigned tech recording diagnostic, got %d: %s", recLegitDiag.Code, recLegitDiag.Body.String())
	}

	// 5. Assigned Tech 1 registers intervention -> MUST BE 201 Created
	reqInterv := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-1/intervention", strings.NewReader(`{"description":"cambio filtro","laborHourCount":1.5,"part":[]}`)), domain.RoleTechnician, "usr-tech1")
	reqInterv.SetPathValue("serviceOrderId", "order-1")
	recInterv := httptest.NewRecorder()
	intervHandler.Register(recInterv, reqInterv)
	if recInterv.Code != http.StatusCreated {
		t.Fatalf("expected 201 for assigned tech registering intervention, got %d", recInterv.Code)
	}
}

// =========================================================================
// GATE 3: State Machine Invariants & Lifecycle Integrity
// =========================================================================
func TestGate3_StateMachineInvariants(t *testing.T) {
	orders := newFgOrderStore(orderFixture(t, "order-ready", domain.StatusReady))
	assignments := newFgAssignmentStore()
	assignments.set(domain.Assignment{
		ID:             "asg-1",
		ServiceOrderID: "order-ready",
		TechnicianID:   "tech-1",
		IsActive:       true,
	})

	techStore := newFgTechStore()
	techStore.add("tech-1", "usr-tech1", "Motor")

	orderUseCase := usecase.NewServiceOrderUseCase(orders, newFakeVehicleStore("vehicle-1"), assignments, techStore, sequentialIDTest(), testClock())
	orderHandler := NewServiceOrderHandler(orderUseCase)

	diagUseCase := usecase.NewDiagnosticUseCase(newFgDiagStore(), orders, assignments, techStore, sequentialIDTest(), testClock())
	diagHandler := NewDiagnosticHandler(diagUseCase)

	intervUseCase := usecase.NewInterventionUseCase(newFgIntervStore(), orders, assignments, techStore, sequentialIDTest(), testClock())
	intervHandler := NewInterventionHandler(intervUseCase)

	// INVARIANT: Advancing to IN_DIAGNOSIS without an assigned technician must be rejected with 422 Unprocessable Entity
	unassignedOrder := orderFixture(t, "order-unassigned", domain.StatusReceived)
	orders.orders["order-unassigned"] = unassignedOrder
	reqAdvanceNoTech := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-unassigned/status", strings.NewReader(`{"status":"IN_DIAGNOSIS"}`)), domain.RoleAdministrator, "usr-admin")
	reqAdvanceNoTech.SetPathValue("serviceOrderId", "order-unassigned")
	recAdvanceNoTech := httptest.NewRecorder()
	orderHandler.Advance(recAdvanceNoTech, reqAdvanceNoTech)
	if recAdvanceNoTech.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 Unprocessable Entity advancing without technician, got %d: %s", recAdvanceNoTech.Code, recAdvanceNoTech.Body.String())
	}

	// INVARIANT: In READY status, adding diagnostics must be rejected with 409 Conflict
	reqDiag := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-ready/diagnostic", strings.NewReader(`{"finding":"falla","componentToRepair":"algo"}`)), domain.RoleTechnician, "usr-tech1")
	reqDiag.SetPathValue("serviceOrderId", "order-ready")
	recDiag := httptest.NewRecorder()
	diagHandler.Record(recDiag, reqDiag)
	if recDiag.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict adding diagnostic in READY state, got %d", recDiag.Code)
	}

	// INVARIANT: In READY status, adding intervention must be rejected with 409 Conflict
	reqInterv := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-ready/intervention", strings.NewReader(`{"description":"reparar","laborHourCount":1,"part":[]}`)), domain.RoleTechnician, "usr-tech1")
	reqInterv.SetPathValue("serviceOrderId", "order-ready")
	recInterv := httptest.NewRecorder()
	intervHandler.Register(recInterv, reqInterv)
	if recInterv.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict adding intervention in READY state, got %d", recInterv.Code)
	}

	// Advance to DELIVERED
	reqAdvance := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-ready/status", strings.NewReader(`{"status":"DELIVERED"}`)), domain.RoleTechnician, "usr-tech1")
	reqAdvance.SetPathValue("serviceOrderId", "order-ready")
	recAdvance := httptest.NewRecorder()
	orderHandler.Advance(recAdvance, reqAdvance)
	if recAdvance.Code != http.StatusOK {
		t.Fatalf("expected 200 advancing to DELIVERED, got %d: %s", recAdvance.Code, recAdvance.Body.String())
	}

	// INVARIANT: From DELIVERED (terminal), advancing status must be rejected
	reqTerminal := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-ready/status", strings.NewReader(`{"status":"IN_REPAIR"}`)), domain.RoleTechnician, "usr-tech1")
	reqTerminal.SetPathValue("serviceOrderId", "order-ready")
	recTerminal := httptest.NewRecorder()
	orderHandler.Advance(recTerminal, reqTerminal)
	if recTerminal.Code != http.StatusConflict && recTerminal.Code != http.StatusForbidden && recTerminal.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected rejection (409, 403 or 422) from terminal DELIVERED status, got %d", recTerminal.Code)
	}
}

// =========================================================================
// GATE 4: Stale UI + Capabilities + Parameterization Verification
// =========================================================================
func TestGate4_StaleUIAndCapabilities(t *testing.T) {
	orders := newFgOrderStore(orderFixture(t, "order-stale", domain.StatusReceived))
	assignments := newFgAssignmentStore()
	assignments.set(domain.Assignment{
		ID:             "asg-1",
		ServiceOrderID: "order-stale",
		TechnicianID:   "tech-1",
		IsActive:       true,
	})

	techStore := newFgTechStore()
	techStore.add("tech-1", "usr-tech1", "Motor")
	techStore.add("tech-2", "usr-tech2", "Frenos")

	orderUseCase := usecase.NewServiceOrderUseCase(orders, newFakeVehicleStore("vehicle-1"), assignments, techStore, sequentialIDTest(), testClock())
	orderHandler := NewServiceOrderHandler(orderUseCase)
	diagUseCase := usecase.NewDiagnosticUseCase(newFgDiagStore(), orders, assignments, techStore, sequentialIDTest(), testClock())
	diagHandler := NewDiagnosticHandler(diagUseCase)

	// 1. Tech 1 queries detail -> receives permissions where canAddDiagnostic == true
	reqDetail := asCallerWithID(httptest.NewRequest(http.MethodGet, "/api/service-order/order-stale", nil), domain.RoleTechnician, "usr-tech1")
	reqDetail.SetPathValue("serviceOrderId", "order-stale")
	recDetail := httptest.NewRecorder()
	orderHandler.Find(recDetail, reqDetail)
	if recDetail.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recDetail.Code)
	}
	var detail serviceOrderResponse
	_ = json.Unmarshal(recDetail.Body.Bytes(), &detail)
	if detail.Permissions == nil || !detail.Permissions.CanAddDiagnostic {
		t.Errorf("expected permissions.canAddDiagnostic to be true for assigned tech")
	}

	// 2. Admin reassigns order-stale to Tech 2 behind Tech 1's back (Stale UI)
	assignments.set(domain.Assignment{ID: "asg-1", ServiceOrderID: "order-stale", TechnicianID: "tech-1", IsActive: false})
	assignments.set(domain.Assignment{ID: "asg-2", ServiceOrderID: "order-stale", TechnicianID: "tech-2", IsActive: true})

	// 3. Tech 1 attempts to submit diagnostic from stale UI -> MUST BE 403 Forbidden
	reqStale := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-stale/diagnostic", strings.NewReader(`{"finding":"stale finding","componentToRepair":"motor"}`)), domain.RoleTechnician, "usr-tech1")
	reqStale.SetPathValue("serviceOrderId", "order-stale")
	recStale := httptest.NewRecorder()
	diagHandler.Record(recStale, reqStale)
	if recStale.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for stale tech after reassignment, got %d", recStale.Code)
	}

	// 4. Tech 2 (newly assigned) submits diagnostic -> MUST BE 201 Created
	reqNew := asCallerWithID(httptest.NewRequest(http.MethodPost, "/api/service-order/order-stale/diagnostic", strings.NewReader(`{"finding":"legit finding","componentToRepair":"frenos"}`)), domain.RoleTechnician, "usr-tech2")
	reqNew.SetPathValue("serviceOrderId", "order-stale")
	recNew := httptest.NewRecorder()
	diagHandler.Record(recNew, reqNew)
	if recNew.Code != http.StatusCreated {
		t.Errorf("expected 201 Created for new assigned tech, got %d: %s", recNew.Code, recNew.Body.String())
	}
}

// =========================================================================
// GATE 5: Concurrency Stress & Race Conditions (go test -race)
// =========================================================================
func TestGate5_ConcurrencyStress(t *testing.T) {
	orders := newFgOrderStore()
	assignments := newFgAssignmentStore()
	techStore := newFgTechStore()
	techStore.add("tech-1", "usr-tech1", "Motor")

	orderUseCase := usecase.NewServiceOrderUseCase(orders, newFakeVehicleStore("vehicle-1"), assignments, techStore, sequentialIDTest(), testClock())
	orderHandler := NewServiceOrderHandler(orderUseCase)

	var wg sync.WaitGroup
	// Concurrently create 20 orders
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := asCaller(httptest.NewRequest(
				http.MethodPost, "/api/service-order",
				strings.NewReader(fmt.Sprintf(`{"vehicleId":"vehicle-1","reportedFailure":"Falla %d"}`, idx)),
			), domain.RoleAdministrator)
			rec := httptest.NewRecorder()
			orderHandler.Create(rec, req)
			if rec.Code != http.StatusCreated {
				t.Errorf("expected 201 on concurrent create, got %d", rec.Code)
			}
		}(i)
	}
	wg.Wait()
}
