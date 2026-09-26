package http

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"workshop/internal/domain"
)

// errorPayload is the only error shape the API returns. The message is written
// in Spanish because the end user reads it.
type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// failure maps a domain error to a status code and a sanitized Spanish
// message.
func failure(writer http.ResponseWriter, err error) {
	status, code, message := classify(err)
	log.Printf("[API ERROR] status=%d code=%s: %v", status, code, err)
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	respond(writer, status, errorPayload{Code: code, Message: message, Detail: detail})
}

func classify(err error) (int, string, string) {
	switch {
	case errors.Is(err, domain.ErrAccountInactive):
		return http.StatusUnauthorized, "account_inactive", "La cuenta de usuario ha sido desactivada por el administrador."
	case errors.Is(err, domain.ErrAuthorizationStateUnavailable):
		return http.StatusServiceUnavailable, "authorization_unavailable", "El servicio de autorizacion no esta disponible temporalmente. Intente mas tarde."
	case errors.Is(err, domain.ErrSelfDeactivation):
		return http.StatusForbidden, "self_deactivation_forbidden", "Un administrador no puede revocar el acceso a su propia cuenta."
	case errors.Is(err, domain.ErrLastAdministrator):
		return http.StatusForbidden, "last_admin_forbidden", "No se puede desactivar al unico administrador activo del sistema."
	case errors.Is(err, domain.ErrTechnicianInactive):
		return http.StatusUnprocessableEntity, "technician_inactive", "El tecnico se encuentra inactivo y no puede recibir ordenes de servicio."
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized", "Usuario o contrasena incorrectos."
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden", "No tiene permiso para realizar esta accion."
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found", "El registro solicitado no existe."
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "conflict", conflictMessage(err)
	case errors.Is(err, domain.ErrInvalidTransition):
		return http.StatusUnprocessableEntity, "invalid_transition", "Transicion de estado no permitida."
	case errors.Is(err, domain.ErrTechnicianRequired):
		return http.StatusUnprocessableEntity, "technician_required", "Se requiere asignar un tecnico responsable antes de iniciar el diagnostico o reparacion."
	case errors.Is(err, domain.ErrWeakPassword):
		return http.StatusBadRequest, "weak_password", "La contraseña no cumple con los requisitos de complejidad (mínimo 8 caracteres, mayúscula, minúscula, número y caracter especial)."
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, "invalid_input", invalidInputMessage(err)
	default:
		return http.StatusInternalServerError, "internal_error", "Ocurrio un error inesperado. Intente de nuevo."
	}
}

// conflictMessage turns the few conflicts the user can act on into a precise
// Spanish sentence, and keeps a neutral one for the rest.
func conflictMessage(err error) string {
	text := err.Error()
	switch {
	case strings.Contains(text, "username") || strings.Contains(text, "uq_user_username"):
		return "El nombre de usuario ya se encuentra registrado."
	case strings.Contains(text, "technician already holds") || strings.Contains(text, "uq_assignment_active_marker"):
		return "El tecnico ya tiene una orden activa."
	case strings.Contains(text, "already has an active technician") || strings.Contains(text, "uq_assignment_active_order_marker"):
		return "La orden ya tiene un tecnico asignado."
	default:
		return "El registro ya existe o entra en conflicto con otro."
	}
}

func invalidInputMessage(err error) string {
	text := err.Error()
	switch {
	case strings.Contains(text, "plate"):
		return "La placa no es valida."
	case strings.Contains(text, "VIN"):
		return "El VIN no es valido."
	case strings.Contains(text, "email"):
		return "El correo no es valido."
	case strings.Contains(text, "labor hour"):
		return "Las horas de trabajo deben ser mayores que cero."
	case strings.Contains(text, "part quantity"):
		return "La cantidad del repuesto debe ser mayor que cero."
	case strings.Contains(text, "coverage in months"):
		return "La cobertura en meses debe ser mayor que cero."
	case strings.Contains(text, "cuerpo de la peticion"):
		return "El cuerpo de la peticion no es valido."
	default:
		return "Los datos enviados no son validos."
	}
}
