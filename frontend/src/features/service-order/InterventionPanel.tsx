/* Intervention panel: the work executed on the vehicle, with a repeatable part
   row, plus the action that issues a warranty over a recorded intervention. */
import { useState } from 'react';

import { ApiError } from '../../services/api_client';
import {
  listIntervention,
  registerIntervention,
} from '../../services/service_order_service';
import type { OrderPermissions, PartUsage } from '../../services/service_order_service';
import { issueWarranty, listWarranty } from '../../services/warranty_service';
import type { WarrantyKind } from '../../services/warranty_service';
import { DataState, ErrorBanner, SuccessBanner } from '../../shared/DataState';
import { ValidityBadge } from '../../shared/StatusBadge';
import { useAsyncData } from '../../shared/useAsyncData';
import { useSession, useToken } from '../../shared/SessionContext';
import { formatDateTime } from '../../shared/format';

interface InterventionPanelProps {
  serviceOrderId: string;
  onChange: () => void;
  orderPermissions?: OrderPermissions;
  permissions?: OrderPermissions;
  isDelivered?: boolean;
}

const EMPTY_PART: PartUsage = { partName: '', quantity: 1 };

export function InterventionPanel({
  serviceOrderId,
  onChange,
  orderPermissions,
  permissions,
  isDelivered,
}: InterventionPanelProps) {
  const token = useToken();
  const { isAdministrator } = useSession();
  const intervention = useAsyncData(
    () => listIntervention(token, serviceOrderId),
    [token, serviceOrderId],
  );
  const warrantyData = useAsyncData(
    () => listWarranty(token, '').catch(() => []),
    [token],
  );
  const [description, setDescription] = useState('');
  const [laborHourCount, setLaborHourCount] = useState('1');
  const [part, setPart] = useState<PartUsage[]>([EMPTY_PART]);
  const [error, setError] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [sending, setSending] = useState(false);
  const [selectedIntervention, setSelectedIntervention] = useState<any | null>(null);
  const [warrantyKind, setWarrantyKind] = useState<WarrantyKind>('PART');
  const [coverageMonths, setCoverageMonths] = useState<number>(3);
  const [issuing, setIssuing] = useState(false);

  const activePermissions = orderPermissions ?? permissions;
  const canAddIntervention =
    !isDelivered &&
    activePermissions !== undefined &&
    activePermissions.canAddIntervention === true;

  const updatePart = (index: number, field: keyof PartUsage, value: string) =>
    setPart((previous) =>
      previous.map((item, position) =>
        position === index
          ? { ...item, [field]: field === 'quantity' ? Number(value) : value }
          : item,
      ),
    );

  const removePart = (index: number) => {
    setPart((previous) => previous.filter((_, position) => position !== index));
  };

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError('');
    setConfirmation('');
    setSending(true);
    try {
      await registerIntervention(
        token,
        serviceOrderId,
        description,
        Number(laborHourCount),
        part.filter((item) => item.partName.trim().length > 0),
      );
      setDescription('');
      setLaborHourCount('1');
      setPart([EMPTY_PART]);
      setConfirmation('Intervencion registrada.');
      intervention.reload();
      warrantyData.reload();
      onChange();
    } catch (failure) {
      setError(
        failure instanceof ApiError ? failure.message : 'No se pudo registrar la intervencion.',
      );
    } finally {
      setSending(false);
    }
  };

  const issue = async (interventionId: string, kind: WarrantyKind, months: number) => {
    setError('');
    setConfirmation('');
    try {
      await issueWarranty(token, interventionId, kind, months);
      setConfirmation(
        `Garantia emitida por ${months} ${months === 1 ? 'mes' : 'meses'} (${kind === 'PART' ? 'Repuesto' : 'Mano de obra'}).`,
      );
      warrantyData.reload();
      intervention.reload();
      onChange();
    } catch (failure) {
      setError(failure instanceof ApiError ? failure.message : 'No se pudo emitir la garantia.');
      throw failure;
    }
  };

  return (
    <section className="card">
      <h3 className="card__title">Intervenciones</h3>
      <ErrorBanner message={error} />
      <SuccessBanner message={confirmation} />

      {canAddIntervention ? (
        <form onSubmit={submit} noValidate>
          <div className="form-grid">
            <div className="field">
              <label className="field__label" htmlFor="description">
                Descripcion
              </label>
              <input
                className="field__input"
                id="description"
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                required
              />
            </div>
            <div className="field">
              <label className="field__label" htmlFor="laborHourCount">
                Horas de trabajo
              </label>
              <input
                className="field__input"
                id="laborHourCount"
                type="number"
                min="0.5"
                step="0.5"
                value={laborHourCount}
                onChange={(event) => setLaborHourCount(event.target.value)}
                required
              />
            </div>
          </div>
          {part.map((item, index) => (
            <div className="form-grid" key={index} style={{ alignItems: 'flex-end' }}>
              <div className="field">
                <label className="field__label" htmlFor={'partName-' + index}>
                  Repuesto
                </label>
                <input
                  className="field__input"
                  id={'partName-' + index}
                  value={item.partName}
                  onChange={(event) => updatePart(index, 'partName', event.target.value)}
                />
              </div>
              <div className="field">
                <label className="field__label" htmlFor={'quantity-' + index}>
                  Cantidad
                </label>
                <input
                  className="field__input"
                  id={'quantity-' + index}
                  type="number"
                  min="1"
                  value={String(item.quantity)}
                  onChange={(event) => updatePart(index, 'quantity', event.target.value)}
                />
              </div>
              {part.length > 1 ? (
                <div className="field" style={{ marginBottom: 'var(--space-3)' }}>
                  <button
                    type="button"
                    className="button button--secondary"
                    onClick={() => removePart(index)}
                    aria-label="Quitar repuesto"
                  >
                    Quitar
                  </button>
                </div>
              ) : null}
            </div>
          ))}
          <div style={{ display: 'flex', gap: '0.75rem', marginTop: '0.5rem' }}>
            <button
              type="button"
              className="button button--secondary"
              onClick={() => setPart((previous) => [...previous, { ...EMPTY_PART }])}
            >
              Agregar repuesto
            </button>
            <button type="submit" className="button button--primary" disabled={sending}>
              {sending ? 'Registrando...' : 'Registrar intervencion'}
            </button>
          </div>
        </form>
      ) : isDelivered ? (
        <p className="state-message">
          Esta orden está entregada (cerrada). El registro de intervenciones está bloqueado.
        </p>
      ) : (
        <p className="state-message">
          No tienes permisos para registrar intervenciones en esta orden o el estado actual no lo permite.
        </p>
      )}

      <DataState
        loading={intervention.loading}
        error={intervention.error}
        empty={(intervention.data ?? []).length === 0}
        emptyMessage="Esta orden aun no tiene intervenciones."
      >
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Descripcion</th>
                <th scope="col">Horas</th>
                <th scope="col">Repuestos</th>
                <th scope="col">Fecha</th>
                <th scope="col">Garantia</th>
              </tr>
            </thead>
            <tbody>
              {(intervention.data ?? []).map((item) => (
                <tr key={item.id}>
                  <td>{item.description}</td>
                  <td>{item.laborHourCount}</td>
                  <td>
                    {item.part.length === 0
                      ? 'Sin repuestos'
                      : item.part.map((used) => used.partName).join(', ')}
                  </td>
                  <td>{formatDateTime(item.performedAt)}</td>
                  <td>
                    {(() => {
                      const associatedWarranty =
                        item.warranty ??
                        (warrantyData.data ?? []).find((w) => w.interventionId === item.id);

                      if (associatedWarranty) {
                        return <ValidityBadge valid={associatedWarranty.valid} />;
                      }

                      if (isAdministrator) {
                        return (
                          <button
                            type="button"
                            className="button button--secondary"
                            onClick={() => {
                              setSelectedIntervention(item);
                              setWarrantyKind(item.part.length > 0 ? 'PART' : 'LABOR');
                              setCoverageMonths(item.part.length > 0 ? 3 : 1);
                            }}
                          >
                            Emitir garantia
                          </button>
                        );
                      }

                      return <span className="state-message">Sin garantía</span>;
                    })()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </DataState>

      {selectedIntervention ? (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="modal-warranty-title"
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(17, 24, 39, 0.6)',
            backdropFilter: 'blur(2px)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1050,
            padding: '1rem',
          }}
        >
          <div
            className="card"
            style={{
              maxWidth: '520px',
              width: '100%',
              backgroundColor: '#ffffff',
              boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.2), 0 10px 10px -5px rgba(0, 0, 0, 0.1)',
              borderRadius: '0.75rem',
              padding: '1.75rem',
              border: '1px solid #e5e7eb',
            }}
          >
            <h3 id="modal-warranty-title" className="card__title" style={{ marginTop: 0, marginBottom: '0.5rem' }}>
              Emitir Póliza de Garantía
            </h3>
            <div
              style={{
                background: '#f8fafc',
                border: '1px solid #e2e8f0',
                borderRadius: '0.5rem',
                padding: '0.75rem 1rem',
                fontSize: '0.875rem',
                color: '#334155',
                marginBottom: '1.25rem',
              }}
            >
              <p style={{ margin: '0 0 0.25rem 0' }}>
                <strong>Intervención:</strong> {selectedIntervention.description}
              </p>
              <p style={{ margin: 0 }}>
                <strong>Repuestos asociados:</strong>{' '}
                {selectedIntervention.part && selectedIntervention.part.length > 0
                  ? selectedIntervention.part.map((p: PartUsage) => `${p.partName} (x${p.quantity})`).join(', ')
                  : 'Sin repuestos registrados (Solo mano de obra)'}
              </p>
            </div>

            <div className="field" style={{ marginBottom: '1.25rem' }}>
              <label className="field__label" htmlFor="modalWarrantyKind">
                Tipo de Cobertura
              </label>
              <select
                id="modalWarrantyKind"
                className="field__input"
                value={warrantyKind}
                onChange={(e) => setWarrantyKind(e.target.value as WarrantyKind)}
              >
                <option value="PART">📦 Repuesto / Producto instalado</option>
                <option value="LABOR">🛠️ Mano de obra (Instalación y Ajuste)</option>
              </select>
            </div>

            <div className="field" style={{ marginBottom: '1.25rem' }}>
              <label className="field__label" htmlFor="modalCoverageMonths">
                Meses de Cobertura
              </label>
              <input
                id="modalCoverageMonths"
                className="field__input"
                type="number"
                min="1"
                max="60"
                value={coverageMonths}
                onChange={(e) => setCoverageMonths(Math.max(1, Number(e.target.value)))}
                required
              />
              <div style={{ display: 'flex', gap: '0.375rem', marginTop: '0.5rem', flexWrap: 'wrap' }}>
                <button
                  type="button"
                  className="button button--secondary"
                  style={{ padding: '0.25rem 0.5rem', fontSize: '0.75rem' }}
                  onClick={() => {
                    setWarrantyKind('LABOR');
                    setCoverageMonths(1);
                  }}
                >
                  1 mes (Mano de obra)
                </button>
                <button
                  type="button"
                  className="button button--secondary"
                  style={{ padding: '0.25rem 0.5rem', fontSize: '0.75rem' }}
                  onClick={() => {
                    setWarrantyKind('PART');
                    setCoverageMonths(3);
                  }}
                >
                  3 meses (Pastillas / Desgaste)
                </button>
                <button
                  type="button"
                  className="button button--secondary"
                  style={{ padding: '0.25rem 0.5rem', fontSize: '0.75rem' }}
                  onClick={() => {
                    setWarrantyKind('PART');
                    setCoverageMonths(6);
                  }}
                >
                  6 meses
                </button>
                <button
                  type="button"
                  className="button button--secondary"
                  style={{ padding: '0.25rem 0.5rem', fontSize: '0.75rem' }}
                  onClick={() => {
                    setWarrantyKind('PART');
                    setCoverageMonths(12);
                  }}
                >
                  12 meses (Motor / Pieza mayor)
                </button>
              </div>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.75rem', marginTop: '1.5rem' }}>
              <button
                type="button"
                className="button button--secondary"
                onClick={() => setSelectedIntervention(null)}
                disabled={issuing}
              >
                Cancelar
              </button>
              <button
                type="button"
                className="button button--primary"
                disabled={issuing || coverageMonths <= 0}
                onClick={async () => {
                  setIssuing(true);
                  try {
                    await issue(selectedIntervention.id, warrantyKind, coverageMonths);
                    setSelectedIntervention(null);
                  } catch {
                    // error handled in issue()
                  } finally {
                    setIssuing(false);
                  }
                }}
              >
                {issuing ? 'Emitiendo...' : 'Confirmar y emitir garantía'}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
