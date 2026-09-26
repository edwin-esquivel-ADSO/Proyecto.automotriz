/* Order detail: the header with the status and the advance buttons, plus the
   four panels that hang from the order. Each panel is its own component. */
import { useState } from 'react';
import { useParams } from 'react-router-dom';

import { ApiError } from '../../services/api_client';
import { advanceServiceOrder, findServiceOrder } from '../../services/service_order_service';
import { AssignmentPanel } from './AssignmentPanel';
import { DiagnosticPanel } from './DiagnosticPanel';
import { InterventionPanel } from './InterventionPanel';
import { StatusHistoryPanel } from './StatusHistoryPanel';
import { DataState, ErrorBanner, SuccessBanner } from '../../shared/DataState';
import { StatusBadge } from '../../shared/StatusBadge';
import { useAsyncData } from '../../shared/useAsyncData';
import { useSession, useToken } from '../../shared/SessionContext';
import { formatDateTime, nextStatus, statusLabel } from '../../shared/format';

export function ServiceOrderDetailPage() {
  const { serviceOrderId = '' } = useParams();
  const token = useToken();
  const { isAdministrator } = useSession();
  const order = useAsyncData(() => findServiceOrder(token, serviceOrderId), [token, serviceOrderId]);
  const [actionError, setActionError] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [sending, setSending] = useState(false);
  const [historyToken, setHistoryToken] = useState(0);

  const current = order.data;
  const following = current ? nextStatus(current.status) : null;
  const hasAssignedTechnician = Boolean(current?.technicianName && current.technicianName !== 'Sin asignar');
  const canAdvance =
    current?.permissions !== undefined
      ? current.permissions.canAdvance
      : Boolean(isAdministrator && following && (following !== 'IN_DIAGNOSIS' && following !== 'IN_REPAIR' || hasAssignedTechnician));

  const advance = async () => {
    if (!following) {
      return;
    }
    setActionError('');
    setConfirmation('');
    setSending(true);
    try {
      await advanceServiceOrder(token, serviceOrderId, following);
      setConfirmation('Estado actualizado a ' + statusLabel(following) + '.');
      order.reload();
      setHistoryToken((previous) => previous + 1);
    } catch (failure) {
      setActionError(
        failure instanceof ApiError ? failure.message : 'No se pudo cambiar el estado.',
      );
    } finally {
      setSending(false);
    }
  };

  return (
    <section>
      <h2 className="screen-title">Detalle de la orden</h2>
      <DataState loading={order.loading} error={order.error} empty={!current}>
        <section className="card">
          <h3 className="card__title">
            {current?.orderNumber} {current ? <StatusBadge status={current.status} /> : null}
          </h3>
          <p>
            <strong>
              {current?.status === 'DELIVERED'
                ? 'Técnico que atendió el servicio:'
                : 'Técnico responsable asignado:'}
            </strong>{' '}
            {current?.technicianName || 'Sin asignar'}
            {current?.technicianIsActive === false ? (
              <span className="badge badge--delivered" style={{ marginLeft: '8px' }}>
                Sin acceso
              </span>
            ) : null}
          </p>
          <p>
            <strong>Falla reportada:</strong> {current?.reportedFailure}
          </p>
          <p className="timeline__date">Ingreso: {formatDateTime(current?.receivedAt ?? '')}</p>
          <ErrorBanner message={actionError} />
          <SuccessBanner message={confirmation} />
          {current?.technicianIsActive === false && current?.status !== 'DELIVERED' ? (
            <p className="state-message" style={{ color: '#b91c1c', fontWeight: 500 }}>
              ⚠️ El técnico responsable asignado tiene el acceso inhabilitado. Reasigne la orden a un técnico activo para continuar los trabajos.
            </p>
          ) : null}
          {!hasAssignedTechnician && following && (following === 'IN_DIAGNOSIS' || following === 'IN_REPAIR') ? (
            <p className="state-message" style={{ color: '#b45309', fontWeight: 500 }}>
              ⚠️ Debe asignar un técnico responsable a la orden antes de iniciar el diagnóstico o la reparación.
            </p>
          ) : null}
          {canAdvance && following ? (
            <button
              type="button"
              className="button button--primary"
              onClick={advance}
              disabled={sending}
            >
              {sending ? 'Actualizando...' : 'Marcar ' + statusLabel(following).toLowerCase()}
            </button>
          ) : null}
          {!following ? <p className="state-message">La orden ya fue entregada.</p> : null}
        </section>

        <div className="panel-stack">
          {isAdministrator && current?.status !== 'DELIVERED' ? (
            <AssignmentPanel
              serviceOrderId={serviceOrderId}
              assignedTechnicianId={current?.assignedTechnicianId}
              technicianIsActive={current?.technicianIsActive}
              onChange={() => order.reload()}
            />
          ) : null}
          <DiagnosticPanel
            serviceOrderId={serviceOrderId}
            orderPermissions={current?.permissions}
            permissions={current?.permissions}
            isDelivered={current?.status === 'DELIVERED'}
            onChange={() => order.reload()}
          />
          <InterventionPanel
            serviceOrderId={serviceOrderId}
            orderPermissions={current?.permissions}
            permissions={current?.permissions}
            isDelivered={current?.status === 'DELIVERED'}
            onChange={() => order.reload()}
          />
          <StatusHistoryPanel serviceOrderId={serviceOrderId} refreshToken={historyToken} />
        </div>
      </DataState>
    </section>
  );
}
