import { useState } from 'react';
import { changePassword } from '../../services/session_service';
import { ApiError } from '../../services/api_client';
import { ErrorBanner, SuccessBanner } from '../../shared/DataState';
import { useSession, useToken } from '../../shared/SessionContext';

export function ChangePasswordModal() {
  const { requiresPasswordChange, onPasswordChanged, signOut } = useSession();
  const token = useToken();

  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [sending, setSending] = useState(false);

  if (!requiresPasswordChange) {
    return null;
  }

  // Password complexity rules
  const hasMinLength = newPassword.length >= 8;
  const hasUpper = /[A-Z]/.test(newPassword);
  const hasLower = /[a-z]/.test(newPassword);
  const hasDigit = /[0-9]/.test(newPassword);
  const hasSpecial = /[^A-Za-z0-9]/.test(newPassword);
  const passwordsMatch = newPassword.length > 0 && newPassword === confirmPassword;

  const isFormValid =
    hasMinLength &&
    hasUpper &&
    hasLower &&
    hasDigit &&
    hasSpecial &&
    passwordsMatch &&
    currentPassword.length > 0;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isFormValid) {
      setError('Por favor complete todos los requisitos de seguridad de la contraseña.');
      return;
    }

    setError('');
    setConfirmation('');
    setSending(true);

    try {
      await changePassword(token, currentPassword, newPassword);
      setConfirmation('Contraseña actualizada exitosamente.');
      setTimeout(() => {
        onPasswordChanged();
      }, 1000);
    } catch (failure) {
      setError(
        failure instanceof ApiError
          ? failure.message
          : 'No se pudo actualizar la contraseña. Verifique su contraseña actual.',
      );
    } finally {
      setSending(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="change-password-title"
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(15, 23, 42, 0.75)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9999,
        padding: '1.25rem',
      }}
    >
      <div
        className="card"
        style={{
          maxWidth: '480px',
          width: '100%',
          backgroundColor: '#ffffff',
          borderRadius: '0.75rem',
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
          padding: '2rem',
        }}
      >
        <div style={{ textAlign: 'center', marginBottom: '1.5rem' }}>
          <div
            style={{
              width: '48px',
              height: '48px',
              margin: '0 auto 0.75rem',
              borderRadius: '50%',
              backgroundColor: '#eff6ff',
              color: '#2563eb',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '1.5rem',
            }}
          >
            🔒
          </div>
          <h2 id="change-password-title" style={{ margin: '0 0 0.5rem 0', fontSize: '1.25rem', fontWeight: 600 }}>
            Cambio Obligatorio de Contraseña
          </h2>
          <p style={{ margin: 0, color: '#64748b', fontSize: '0.875rem', lineHeight: 1.4 }}>
            Por políticas de seguridad, las cuentas con credenciales iniciales deben establecer una nueva clave personal antes de continuar.
          </p>
        </div>

        <ErrorBanner message={error} />
        <SuccessBanner message={confirmation} />

        <form onSubmit={handleSubmit} noValidate>
          <div className="field" style={{ marginBottom: '1rem' }}>
            <label className="field__label" htmlFor="currentPassword">
              Contraseña actual
            </label>
            <input
              id="currentPassword"
              type="password"
              className="field__input"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              required
              autoFocus
            />
          </div>

          <div className="field" style={{ marginBottom: '1rem' }}>
            <label className="field__label" htmlFor="newPassword">
              Nueva contraseña
            </label>
            <input
              id="newPassword"
              type="password"
              className="field__input"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
            />
          </div>

          <div className="field" style={{ marginBottom: '1.25rem' }}>
            <label className="field__label" htmlFor="confirmPassword">
              Confirmar nueva contraseña
            </label>
            <input
              id="confirmPassword"
              type="password"
              className="field__input"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
            />
          </div>

          {/* Complexity Checklist */}
          <div
            style={{
              backgroundColor: '#f8fafc',
              border: '1px solid #e2e8f0',
              borderRadius: '0.5rem',
              padding: '0.75rem 1rem',
              marginBottom: '1.5rem',
              fontSize: '0.8125rem',
              color: '#475569',
            }}
          >
            <strong style={{ display: 'block', marginBottom: '0.375rem', color: '#1e293b' }}>
              Requisitos de la contraseña:
            </strong>
            <ul style={{ listStyle: 'none', margin: 0, padding: 0 }}>
              <li style={{ color: hasMinLength ? '#16a34a' : '#94a3b8', marginBottom: '0.25rem' }}>
                {hasMinLength ? '✓' : '○'} Mínimo 8 caracteres
              </li>
              <li style={{ color: hasUpper ? '#16a34a' : '#94a3b8', marginBottom: '0.25rem' }}>
                {hasUpper ? '✓' : '○'} Al menos una mayúscula (A-Z)
              </li>
              <li style={{ color: hasLower ? '#16a34a' : '#94a3b8', marginBottom: '0.25rem' }}>
                {hasLower ? '✓' : '○'} Al menos una minúscula (a-z)
              </li>
              <li style={{ color: hasDigit ? '#16a34a' : '#94a3b8', marginBottom: '0.25rem' }}>
                {hasDigit ? '✓' : '○'} Al menos un número (0-9)
              </li>
              <li style={{ color: hasSpecial ? '#16a34a' : '#94a3b8', marginBottom: '0.25rem' }}>
                {hasSpecial ? '✓' : '○'} Al menos un caracter especial (!@#$%*...)
              </li>
              <li style={{ color: passwordsMatch ? '#16a34a' : '#94a3b8' }}>
                {passwordsMatch ? '✓' : '○'} Las contraseñas coinciden
              </li>
            </ul>
          </div>

          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <button
              type="button"
              className="button button--secondary"
              onClick={signOut}
              disabled={sending}
            >
              Cerrar sesión
            </button>
            <button
              type="submit"
              className="button button--primary"
              disabled={sending || !isFormValid}
            >
              {sending ? 'Guardando...' : 'Actualizar contraseña'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
