/*
 * Standalone mock API engine for Vercel and offline deployments.
 * Enables 100% functionality without an external database or backend server.
 * Persists changes in localStorage so reloads and user flows work seamlessly.
 */

import type { Customer, NewCustomer } from './customer_service';
import type { Vehicle, NewVehicle } from './vehicle_service';
import type { Technician } from './technician_service';
import type {
  ServiceOrder,
  ServiceOrderStatus,
  StatusTransition,
  Assignment,
  Diagnostic,
  Intervention,
  PartUsage,
} from './service_order_service';
import type { Warranty, WarrantyKind } from './warranty_service';
import type { Dashboard } from './dashboard_service';
import type { Timeline, TimelineEntry } from './timeline_service';
import type { Session } from './session_service';

const DB_KEY = 'workshop.demo.db.v2';

export interface MockUser {
  id: string;
  username: string;
  password: string;
  fullName: string;
  role: 'ADMINISTRATOR' | 'TECHNICIAN';
  requiresPasswordChange: boolean;
}

export function getDefaultUsers(): MockUser[] {
  return [
    {
      id: '11111111-1111-4111-8111-111111111111',
      username: 'admin',
      password: 'Admin2026*',
      fullName: 'Administrador del taller',
      role: 'ADMINISTRATOR',
      requiresPasswordChange: false,
    },
    {
      id: '22222222-2222-4222-8222-222222222222',
      username: 'jperez',
      password: 'Admin2026*',
      fullName: 'Juan Perez',
      role: 'TECHNICIAN',
      requiresPasswordChange: false,
    },
    {
      id: '33333333-3333-4333-8333-333333333333',
      username: 'lramirez',
      password: 'Admin2026*',
      fullName: 'Laura Ramirez',
      role: 'TECHNICIAN',
      requiresPasswordChange: false,
    },
  ];
}

interface MockDatabase {
  users?: MockUser[];
  customers: Customer[];
  vehicles: Vehicle[];
  technicians: Technician[];
  orders: ServiceOrder[];
  assignments: Assignment[];
  diagnostics: Diagnostic[];
  interventions: Intervention[];
  warranties: Warranty[];
  transitions: StatusTransition[];
}

function getInitialData(): MockDatabase {
  const customer1: Customer = {
    id: 'c1000000-0000-4000-8000-000000000001',
    fullName: 'Carlos Mendoza',
    documentNumber: '1020304050',
    phone: '3101234567',
    email: 'carlos.mendoza@email.com',
    createdAt: '2026-03-01T08:00:00Z',
  };

  const customer2: Customer = {
    id: 'c2000000-0000-4000-8000-000000000002',
    fullName: 'Ana María Gómez',
    documentNumber: '1098765432',
    phone: '3159876543',
    email: 'ana.gomez@email.com',
    createdAt: '2026-03-05T09:30:00Z',
  };

  const vehicle1: Vehicle = {
    id: 'v1000000-0000-4000-8000-000000000001',
    customerId: customer1.id,
    ownerName: customer1.fullName,
    plate: 'ABC123',
    vin: '1HGCR2F83HA000001',
    brand: 'Toyota',
    model: 'Corolla',
    modelYear: 2022,
    createdAt: '2026-03-01T08:15:00Z',
  };

  const vehicle2: Vehicle = {
    id: 'v2000000-0000-4000-8000-000000000002',
    customerId: customer2.id,
    ownerName: customer2.fullName,
    plate: 'XYZ789',
    vin: '3N1AB7AP4HY000002',
    brand: 'Chevrolet',
    model: 'Onix',
    modelYear: 2021,
    createdAt: '2026-03-05T09:45:00Z',
  };

  const tech1: Technician = {
    id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1',
    userId: '22222222-2222-4222-8222-222222222222',
    fullName: 'Juan Perez',
    specialty: 'Motor y transmision',
    isActive: true,
    busy: true,
    canReceiveAssignment: false,
    activeOrderId: 'o1000000-0000-4000-8000-000000000001',
    activeOrderNumber: 'OS-0001',
    activeVehiclePlate: 'ABC123',
  };

  const tech2: Technician = {
    id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2',
    userId: '33333333-3333-4333-8333-333333333333',
    fullName: 'Laura Ramirez',
    specialty: 'Frenos y suspension',
    isActive: true,
    busy: false,
    canReceiveAssignment: true,
    activeOrderId: '',
    activeOrderNumber: '',
    activeVehiclePlate: '',
  };

  const order1: ServiceOrder = {
    id: 'o1000000-0000-4000-8000-000000000001',
    orderNumber: 'OS-0001',
    vehicleId: vehicle1.id,
    vehiclePlate: vehicle1.plate,
    technicianName: tech1.fullName,
    reportedFailure: 'Ruido anormal en el compartimento de motor y perdida leve de potencia al acelerar.',
    status: 'IN_DIAGNOSIS',
    receivedAt: '2026-03-10T08:30:00Z',
    updatedAt: '2026-03-10T09:15:00Z',
  };

  const assignment1: Assignment = {
    id: 'a1000000-0000-4000-8000-000000000001',
    serviceOrderId: order1.id,
    technicianId: tech1.id,
    isActive: true,
    assignedAt: '2026-03-10T08:45:00Z',
  };

  const diagnostic1: Diagnostic = {
    id: 'd1000000-0000-4000-8000-000000000001',
    serviceOrderId: order1.id,
    technicianId: tech1.id,
    finding: 'Fuga en empaque de valvulas y bujias con desgaste prematuro.',
    componentToRepair: 'Empaque de tapa de valvulas y juego de bujias de iridio.',
    createdAt: '2026-03-10T09:15:00Z',
  };

  const transition1: StatusTransition = {
    id: 't1000000-0000-4000-8000-000000000001',
    fromStatus: 'RECEIVED',
    toStatus: 'IN_DIAGNOSIS',
    changedByName: 'Administrador del taller',
    changedAt: '2026-03-10T09:15:00Z',
  };

  return {
    users: getDefaultUsers(),
    customers: [customer1, customer2],
    vehicles: [vehicle1, vehicle2],
    technicians: [tech1, tech2],
    orders: [order1],
    assignments: [assignment1],
    diagnostics: [diagnostic1],
    interventions: [],
    warranties: [],
    transitions: [transition1],
  };
}

function loadDB(): MockDatabase {
  try {
    const raw = localStorage.getItem(DB_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as MockDatabase;
      if (!parsed.users || parsed.users.length === 0) {
        parsed.users = getDefaultUsers();
        saveDB(parsed);
      }
      return parsed;
    }
  } catch {
    // Local storage not accessible
  }
  const initial = getInitialData();
  saveDB(initial);
  return initial;
}

function saveDB(data: MockDatabase): void {
  try {
    localStorage.setItem(DB_KEY, JSON.stringify(data));
  } catch {
    // Storage full or unavailable
  }
}

function jsonResponse(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function errorResponse(status: number, code: string, message: string): Response {
  return new Response(JSON.stringify({ code, message }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

export function setupBrowserMockApi(): void {
  if (typeof window === 'undefined') return;

  const originalFetch = window.fetch;

  window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    let urlString = '';
    if (typeof input === 'string') {
      urlString = input;
    } else if (input instanceof URL) {
      urlString = input.toString();
    } else if (input && typeof (input as Request).url === 'string') {
      urlString = (input as Request).url;
    }

    // Only intercept local /api/ routes
    const isApiCall =
      urlString.startsWith('/api') ||
      urlString.includes('/api/') ||
      (urlString.startsWith(window.location.origin) && urlString.includes('/api/'));

    if (!isApiCall) {
      return originalFetch(input, init);
    }

    // If an external backend is explicitly configured and not localhost, try real fetch first
    const customApiUrl = (import.meta.env.VITE_API_URL as string | undefined)?.trim();
    if (customApiUrl && !customApiUrl.startsWith('/api') && !customApiUrl.includes('localhost')) {
      try {
        const response = await originalFetch(input, init);
        if (response.ok || response.status < 500) {
          return response;
        }
      } catch {
        // Fallback to local mock if remote server is unreachable
      }
    }

    // Parse path and query
    let path = urlString;
    if (path.startsWith(window.location.origin)) {
      path = path.slice(window.location.origin.length);
    }
    const [pathname, search = ''] = path.split('?');
    const searchParams = new URLSearchParams(search);
    const method = (init?.method || 'GET').toUpperCase();

    let body: any = null;
    if (init?.body && typeof init.body === 'string') {
      try {
        body = JSON.parse(init.body);
      } catch {
        body = null;
      }
    }

    // Extract Authorization header
    let authHeader = '';
    if (init?.headers) {
      if (init.headers instanceof Headers) {
        authHeader = init.headers.get('Authorization') || '';
      } else if (Array.isArray(init.headers)) {
        const found = init.headers.find(([k]) => k.toLowerCase() === 'authorization');
        authHeader = found ? found[1] : '';
      } else if (typeof init.headers === 'object') {
        const headersObj = init.headers as Record<string, string>;
        authHeader = headersObj['Authorization'] || headersObj['authorization'] || '';
      }
    }
    const token = authHeader.replace(/^Bearer\s+/i, '').trim();
    const isCallerAdmin = token === 'token-admin-session-mock';
    const isCallerJPerez = token === 'token-jperez-session-mock';
    const isCallerLRamirez = token === 'token-lramirez-session-mock';
    const isCallerTechnician = isCallerJPerez || isCallerLRamirez;
    const callerTechId = isCallerJPerez
      ? 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1'
      : isCallerLRamirez
      ? 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2'
      : null;

    const db = loadDB();

    // 1. Session / Auth
    if (pathname === '/api/session' && method === 'POST') {
      const username = body?.username?.trim().toLowerCase();
      const password = body?.password?.trim();

      const validSharedPass = ['Admin2026*', 'admin2026*'];
      const validJPerezPass = [...validSharedPass, 'JPerez2026*', 'jperez2026*'];
      const validLRamirezPass = [...validSharedPass, 'LRamirez2026*', 'lramirez2026*'];

      const user = db.users?.find((u) => u.username.toLowerCase() === username);

      if (username === 'admin' && (validSharedPass.includes(password) || user?.password === password)) {
        const session: Session = {
          token: 'token-admin-session-mock',
          expiresAt: new Date(Date.now() + 8 * 3600 * 1000).toISOString(),
          userId: '11111111-1111-4111-8111-111111111111',
          username: 'admin',
          fullName: 'Administrador del taller',
          role: 'ADMINISTRATOR',
          requiresPasswordChange: user ? user.requiresPasswordChange : false,
        };
        return jsonResponse(session);
      }
      if (username === 'jperez' && (validJPerezPass.includes(password) || user?.password === password)) {
        const session: Session = {
          token: 'token-jperez-session-mock',
          expiresAt: new Date(Date.now() + 8 * 3600 * 1000).toISOString(),
          userId: '22222222-2222-4222-8222-222222222222',
          username: 'jperez',
          fullName: 'Juan Perez',
          role: 'TECHNICIAN',
          requiresPasswordChange: user ? user.requiresPasswordChange : false,
        };
        return jsonResponse(session);
      }
      if (username === 'lramirez' && (validLRamirezPass.includes(password) || user?.password === password)) {
        const session: Session = {
          token: 'token-lramirez-session-mock',
          expiresAt: new Date(Date.now() + 8 * 3600 * 1000).toISOString(),
          userId: '33333333-3333-4333-8333-333333333333',
          username: 'lramirez',
          fullName: 'Laura Ramirez',
          role: 'TECHNICIAN',
          requiresPasswordChange: user ? user.requiresPasswordChange : false,
        };
        return jsonResponse(session);
      }
      if (user && user.password === password) {
        const session: Session = {
          token: `token-${user.id}-session-mock`,
          expiresAt: new Date(Date.now() + 8 * 3600 * 1000).toISOString(),
          userId: user.id,
          username: user.username,
          fullName: user.fullName,
          role: user.role,
          requiresPasswordChange: user.requiresPasswordChange,
        };
        return jsonResponse(session);
      }
      return errorResponse(401, 'unauthorized', 'Credenciales invalidas. Verifique su usuario y contrasena.');
    }

    // 1.1 Change Password
    if (pathname === '/api/session/change-password' && method === 'POST') {
      if (!token) {
        return errorResponse(401, 'unauthorized', 'Sesión no válida o expirada.');
      }
      const { currentPassword, newPassword } = body || {};
      if (!currentPassword || !newPassword) {
        return errorResponse(400, 'bad_request', 'La contraseña actual y la nueva son requeridas.');
      }
      const hasLength = newPassword.length >= 8;
      const hasUpper = /[A-Z]/.test(newPassword);
      const hasLower = /[a-z]/.test(newPassword);
      const hasDigit = /[0-9]/.test(newPassword);
      const hasSpecial = /[^A-Za-z0-9]/.test(newPassword);
      if (!hasLength || !hasUpper || !hasLower || !hasDigit || !hasSpecial) {
        return errorResponse(
          400,
          'weak_password',
          'La contraseña no cumple con los requisitos de complejidad (mínimo 8 caracteres, mayúscula, minúscula, número y caracter especial).',
        );
      }

      const user = db.users?.find((u) =>
        (isCallerAdmin && u.username === 'admin') ||
        (isCallerJPerez && u.username === 'jperez') ||
        (isCallerLRamirez && u.username === 'lramirez') ||
        token.includes(u.id),
      );
      if (!user) {
        return errorResponse(401, 'unauthorized', 'Usuario de la sesión no encontrado.');
      }
      const validPasswords = [user.password, 'Admin2026*'];
      if (user.username === 'jperez') validPasswords.push('JPerez2026*');
      if (user.username === 'lramirez') validPasswords.push('LRamirez2026*');
      if (!validPasswords.includes(currentPassword)) {
        return errorResponse(401, 'unauthorized', 'Contraseña actual incorrecta.');
      }

      user.password = newPassword;
      user.requiresPasswordChange = false;
      saveDB(db);
      return jsonResponse({ ok: true, message: 'Contraseña actualizada exitosamente.' });
    }

    if (pathname === '/api/session/logout' && method === 'POST') {
      return jsonResponse({ ok: true });
    }

    if (pathname === '/api/health') {
      return jsonResponse({ status: 'ok' });
    }

    // 2. Customers (Administrator only)
    if (pathname === '/api/customer') {
      if (isCallerTechnician) {
        return errorResponse(403, 'forbidden', 'No tiene permiso para acceder a clientes.');
      }
      if (method === 'GET') {
        return jsonResponse(db.customers);
      }
      if (method === 'POST') {
        const { fullName, documentNumber, phone, email } = body as NewCustomer;
        if (!fullName || !documentNumber) {
          return errorResponse(400, 'bad_request', 'Nombre y documento son obligatorios.');
        }
        if (/<[a-z][\s\S]*>/i.test(fullName) || /<script/i.test(fullName) || /<script/i.test(documentNumber)) {
          return errorResponse(400, 'invalid_input', 'Los datos no pueden contener etiquetas HTML o scripts.');
        }
        if (db.customers.some((c) => c.documentNumber === documentNumber)) {
          return errorResponse(409, 'conflict', 'Ya existe un cliente con ese numero de documento.');
        }
        const newCustomer: Customer = {
          id: crypto.randomUUID ? crypto.randomUUID() : 'c-' + Date.now(),
          fullName,
          documentNumber,
          phone: phone || '',
          email: email || '',
          createdAt: new Date().toISOString(),
        };
        db.customers.unshift(newCustomer);
        saveDB(db);
        return jsonResponse(newCustomer, 201);
      }
    }

    // 3. Vehicles (Administrator only)
    if (pathname === '/api/vehicle') {
      if (isCallerTechnician) {
        return errorResponse(403, 'forbidden', 'No tiene permiso para acceder a vehiculos.');
      }
      if (method === 'GET') {
        return jsonResponse(db.vehicles);
      }
      if (method === 'POST') {
        const { customerId, plate, vin, brand, model, modelYear } = body as NewVehicle;
        if (!customerId || !plate || !vin) {
          return errorResponse(400, 'bad_request', 'Todos los datos del vehiculo son obligatorios.');
        }
        if (db.vehicles.some((v) => v.plate.toUpperCase() === plate.toUpperCase())) {
          return errorResponse(409, 'conflict', 'Ya existe un vehiculo con esa placa.');
        }
        const owner = db.customers.find((c) => c.id === customerId);
        const newVehicle: Vehicle = {
          id: crypto.randomUUID ? crypto.randomUUID() : 'v-' + Date.now(),
          customerId,
          ownerName: owner ? owner.fullName : 'Propietario',
          plate: plate.toUpperCase(),
          vin: vin.toUpperCase(),
          brand,
          model,
          modelYear: Number(modelYear),
          createdAt: new Date().toISOString(),
        };
        db.vehicles.unshift(newVehicle);
        saveDB(db);
        return jsonResponse(newVehicle, 201);
      }
    }

    // 4. Vehicle Timeline (Administrator only)
    const timelineMatch = pathname.match(/^\/api\/vehicle\/([^/]+)\/timeline$/);
    if (timelineMatch && method === 'GET') {
      if (isCallerTechnician) {
        return errorResponse(403, 'forbidden', 'No tiene permiso para ver la linea de tiempo del vehiculo.');
      }
      const vehicleId = timelineMatch[1];
      const vehicle = db.vehicles.find((v) => v.id === vehicleId);
      if (!vehicle) {
        return errorResponse(404, 'not_found', 'Vehiculo no encontrado.');
      }
      const entries: TimelineEntry[] = [];
      const vehicleOrders = db.orders.filter((o) => o.vehicleId === vehicleId);

      for (const ord of vehicleOrders) {
        entries.push({
          kind: 'ORDER',
          occurredAt: ord.receivedAt,
          title: `Apertura de orden ${ord.orderNumber}`,
          description: ord.reportedFailure,
          reference: ord.orderNumber,
        });

        const diag = db.diagnostics.find((d) => d.serviceOrderId === ord.id);
        if (diag) {
          entries.push({
            kind: 'DIAGNOSTIC',
            occurredAt: diag.createdAt,
            title: `Diagnostico tecnico (${ord.orderNumber})`,
            description: `${diag.finding} | Componente: ${diag.componentToRepair}`,
            reference: ord.orderNumber,
          });
        }

        const intervs = db.interventions.filter((i) => i.serviceOrderId === ord.id);
        for (const itv of intervs) {
          const partsStr = itv.part.map((p) => `${p.partName} (x${p.quantity})`).join(', ');
          entries.push({
            kind: 'INTERVENTION',
            occurredAt: itv.performedAt,
            title: `Intervencion: ${itv.description}`,
            description: `${itv.laborHourCount} horas de labor. ${partsStr ? 'Repuestos: ' + partsStr : ''}`,
            reference: ord.orderNumber,
          });
        }
      }

      entries.sort((a, b) => new Date(b.occurredAt).getTime() - new Date(a.occurredAt).getTime());

      const timeline: Timeline = { vehicle, entry: entries };
      return jsonResponse(timeline);
    }

    // 5. Technicians (Administrator only)
    if (pathname === '/api/technician') {
      if (isCallerTechnician) {
        return errorResponse(403, 'forbidden', 'No tiene permiso para ver la lista de tecnicos.');
      }
      if (method === 'GET') {
        const updatedTechnicians = db.technicians.map((tech) => {
          const isActive = tech.isActive !== false;
          const activeAssignment = db.assignments.find((a) => {
            if (a.technicianId !== tech.id || !a.isActive) return false;
            const relatedOrder = db.orders.find((o) => o.id === a.serviceOrderId);
            return relatedOrder && relatedOrder.status !== 'DELIVERED';
          });
          const busy = Boolean(activeAssignment);
          const canReceiveAssignment = isActive && !busy;
          if (activeAssignment) {
            const relatedOrder = db.orders.find((o) => o.id === activeAssignment.serviceOrderId);
            return {
              ...tech,
              isActive,
              busy: true,
              canReceiveAssignment,
              activeOrderId: relatedOrder?.id || '',
              activeOrderNumber: relatedOrder?.orderNumber || '',
              activeVehiclePlate: relatedOrder?.vehiclePlate || '',
            };
          }
          return {
            ...tech,
            isActive,
            busy: false,
            canReceiveAssignment,
            activeOrderId: '',
            activeOrderNumber: '',
            activeVehiclePlate: '',
          };
        });
        return jsonResponse(updatedTechnicians);
      }
      if (method === 'POST') {
        const { fullName, username, password, specialty } = body || {};
        if (!fullName || !username || !password || !specialty) {
          return errorResponse(400, 'bad_request', 'Todos los campos son obligatorios.');
        }
        const hasLength = password.length >= 8;
        const hasUpper = /[A-Z]/.test(password);
        const hasLower = /[a-z]/.test(password);
        const hasDigit = /[0-9]/.test(password);
        const hasSpecial = /[^A-Za-z0-9]/.test(password);
        if (!hasLength || !hasUpper || !hasLower || !hasDigit || !hasSpecial) {
          return errorResponse(
            400,
            'weak_password',
            'La contraseña no cumple con los requisitos de complejidad (mínimo 8 caracteres, mayúscula, minúscula, número y caracter especial).',
          );
        }
        const techUserId = crypto.randomUUID ? crypto.randomUUID() : 'u-' + Date.now();
        const newTech: Technician = {
          id: crypto.randomUUID ? crypto.randomUUID() : 't-' + Date.now(),
          userId: techUserId,
          fullName,
          specialty,
          isActive: true,
          busy: false,
          canReceiveAssignment: true,
          activeOrderId: '',
          activeOrderNumber: '',
          activeVehiclePlate: '',
        };
        db.technicians.push(newTech);
        if (!db.users) db.users = getDefaultUsers();
        db.users.push({
          id: techUserId,
          username: username.trim().toLowerCase(),
          password,
          fullName,
          role: 'TECHNICIAN',
          requiresPasswordChange: true,
        });
        saveDB(db);
        return jsonResponse(newTech, 201);
      }
    }

    const techAccessMatch = pathname.match(/^\/api\/technician\/([^/]+)\/access$/);
    if (techAccessMatch && method === 'PATCH') {
      if (isCallerTechnician) {
        return errorResponse(403, 'forbidden', 'Solo el administrador puede modificar el acceso de tecnicos.');
      }
      const technicianId = techAccessMatch[1];
      const tech = db.technicians.find((t) => t.id === technicianId);
      if (!tech) {
        return errorResponse(404, 'not_found', 'Tecnico no encontrado.');
      }
      const active = Boolean(body?.active);
      tech.isActive = active;
      const activeAssignment = db.assignments.find((a) => {
        if (a.technicianId !== tech.id || !a.isActive) return false;
        const relatedOrder = db.orders.find((o) => o.id === a.serviceOrderId);
        return relatedOrder && relatedOrder.status !== 'DELIVERED';
      });
      tech.busy = Boolean(activeAssignment);
      tech.canReceiveAssignment = active && !tech.busy;
      saveDB(db);
      return jsonResponse(tech);
    }

    // 6. Service Orders List & Create
    if (pathname === '/api/service-order') {
      if (method === 'GET') {
        const filterStatus = searchParams.get('status');
        let result = db.orders;
        if (isCallerTechnician && callerTechId) {
          const assignedIds = new Set(
            db.assignments.filter((a) => a.technicianId === callerTechId && a.isActive).map((a) => a.serviceOrderId),
          );
          result = result.filter((o) => assignedIds.has(o.id));
        }
        if (filterStatus) {
          result = result.filter((o) => o.status === filterStatus);
        }
        return jsonResponse(result);
      }
      if (method === 'POST') {
        const { vehicleId, reportedFailure } = body || {};
        const vehicle = db.vehicles.find((v) => v.id === vehicleId);
        if (!vehicle) {
          return errorResponse(400, 'bad_request', 'Vehiculo no encontrado.');
        }

        const nextNum = String(db.orders.length + 1).padStart(4, '0');
        const orderNumber = `OS-${nextNum}`;
        const newOrder: ServiceOrder = {
          id: crypto.randomUUID ? crypto.randomUUID() : 'o-' + Date.now(),
          orderNumber,
          vehicleId,
          vehiclePlate: vehicle.plate,
          technicianName: '',
          reportedFailure: reportedFailure || 'Revision general',
          status: 'RECEIVED',
          receivedAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };

        db.orders.unshift(newOrder);
        db.transitions.push({
          id: crypto.randomUUID ? crypto.randomUUID() : 't-' + Date.now(),
          fromStatus: 'RECEIVED',
          toStatus: 'RECEIVED',
          changedByName: 'Recepcion',
          changedAt: new Date().toISOString(),
        });
        saveDB(db);
        return jsonResponse(newOrder, 201);
      }
    }

    // 7. Single Service Order Details
    const orderMatch = pathname.match(/^\/api\/service-order\/([^/]+)$/);
    if (orderMatch && method === 'GET') {
      const orderId = orderMatch[1];
      const order = db.orders.find((o) => o.id === orderId);
      if (!order) {
        return errorResponse(404, 'not_found', 'Orden no encontrada.');
      }
      const activeAss = db.assignments.find((a) => a.serviceOrderId === orderId && a.isActive);
      const assignedTech = activeAss ? db.technicians.find((t) => t.id === activeAss.technicianId) : null;
      if (isCallerTechnician && callerTechId) {
        const isAssigned = activeAss && activeAss.technicianId === callerTechId;
        if (!isAssigned) {
          return errorResponse(403, 'forbidden', 'Esta orden no esta asignada a su usuario.');
        }
      }

      const isAssigned = Boolean(activeAss);
      const isTechActive = assignedTech ? assignedTech.isActive !== false : true;
      const isAssignedToCaller = Boolean(activeAss && activeAss.technicianId === callerTechId);
      const isDelivered = order.status === 'DELIVERED';

      const permissions = {
        canAdvance: !isDelivered && (
          isCallerAdmin
            ? (order.status === 'RECEIVED' ? isAssigned && isTechActive : true)
            : (isAssignedToCaller && isTechActive)
        ),
        canAddDiagnostic: !isDelivered && isAssignedToCaller && isTechActive && order.status === 'IN_DIAGNOSIS' && !db.diagnostics.some((d) => d.serviceOrderId === orderId),
        canAddIntervention: !isDelivered && isAssignedToCaller && isTechActive && order.status === 'IN_REPAIR',
        canAssign: !isDelivered && isCallerAdmin,
      };

      const enrichedOrder = {
        ...order,
        assignedTechnicianId: activeAss?.technicianId || '',
        technicianIsActive: isTechActive,
        permissions,
      };

      return jsonResponse(enrichedOrder);
    }

    // 8. Service Order Status Advance
    const statusMatch = pathname.match(/^\/api\/service-order\/([^/]+)\/status$/);
    if (statusMatch && method === 'POST') {
      const orderId = statusMatch[1];
      const nextStatus = body?.status as ServiceOrderStatus;
      const order = db.orders.find((o) => o.id === orderId);
      if (!order) {
        return errorResponse(404, 'not_found', 'Orden no encontrada.');
      }
      if (order.status === 'DELIVERED') {
        return errorResponse(403, 'forbidden', 'No se puede modificar una orden que ya fue entregada.');
      }
      if (isCallerTechnician && callerTechId) {
        const activeAss = db.assignments.find((a) => a.serviceOrderId === orderId && a.isActive);
        if (!activeAss || activeAss.technicianId !== callerTechId) {
          return errorResponse(403, 'forbidden', 'Solo el tecnico asignado puede cambiar el estado de la orden.');
        }
      }

      const prevStatus = order.status;
      order.status = nextStatus;
      order.updatedAt = new Date().toISOString();

      // If delivered, release active assignment
      if (nextStatus === 'DELIVERED') {
        const activeAss = db.assignments.find((a) => a.serviceOrderId === orderId && a.isActive);
        if (activeAss) {
          activeAss.isActive = false;
        }
      }

      db.transitions.push({
        id: crypto.randomUUID ? crypto.randomUUID() : 't-' + Date.now(),
        fromStatus: prevStatus,
        toStatus: nextStatus,
        changedByName: isCallerAdmin ? 'Administrador del taller' : 'Tecnico asignado',
        changedAt: new Date().toISOString(),
      });

      saveDB(db);
      return jsonResponse(order);
    }

    // 9. Status Transitions
    const transitionsMatch = pathname.match(/^\/api\/service-order\/([^/]+)\/transition$/);
    if (transitionsMatch && method === 'GET') {
      const orderId = transitionsMatch[1];
      const history = (db.transitions as (StatusTransition & { serviceOrderId?: string })[])
        .filter((t) => !t.serviceOrderId || t.serviceOrderId === orderId);
      return jsonResponse(history);
    }

    // 10. Assignment
    const assignmentMatch = pathname.match(/^\/api\/service-order\/([^/]+)\/assignment$/);
    if (assignmentMatch) {
      const orderId = assignmentMatch[1];
      if (method === 'GET') {
        const assignment = db.assignments.find((a) => a.serviceOrderId === orderId && a.isActive);
        if (!assignment) {
          return errorResponse(404, 'not_found', 'No hay tecnico asignado.');
        }
        return jsonResponse(assignment);
      }
      if (method === 'POST') {
        if (isCallerTechnician) {
          return errorResponse(403, 'forbidden', 'Solo el jefe de taller puede asignar tecnicos.');
        }
        const order = db.orders.find((o) => o.id === orderId);
        if (order?.status === 'DELIVERED') {
          return errorResponse(403, 'forbidden', 'No se puede reasignar una orden entregada.');
        }
        const { technicianId } = body || {};
        const tech = db.technicians.find((t) => t.id === technicianId);
        if (!tech) {
          return errorResponse(404, 'not_found', 'Tecnico no encontrado.');
        }

        // Deactivate previous active assignment on this order
        db.assignments.forEach((a) => {
          if (a.serviceOrderId === orderId) a.isActive = false;
        });

        const newAssignment: Assignment = {
          id: crypto.randomUUID ? crypto.randomUUID() : 'a-' + Date.now(),
          serviceOrderId: orderId,
          technicianId,
          isActive: true,
          assignedAt: new Date().toISOString(),
        };
        db.assignments.push(newAssignment);

        // Update technician name on order
        if (order) {
          order.technicianName = tech.fullName;
          order.updatedAt = new Date().toISOString();
        }

        saveDB(db);
        return jsonResponse(newAssignment);
      }
    }

    // 11. Diagnostic
    const diagnosticMatch = pathname.match(/^\/api\/service-order\/([^/]+)\/diagnostic$/);
    if (diagnosticMatch) {
      const orderId = diagnosticMatch[1];
      if (method === 'GET') {
        const diagnostic = db.diagnostics.find((d) => d.serviceOrderId === orderId);
        if (!diagnostic) {
          return errorResponse(404, 'not_found', 'Sin diagnostico.');
        }
        return jsonResponse(diagnostic);
      }
      if (method === 'POST') {
        const order = db.orders.find((o) => o.id === orderId);
        if (order?.status === 'DELIVERED') {
          return errorResponse(409, 'conflict', 'No se puede registrar diagnostico en una orden entregada.');
        }
        const activeAss = db.assignments.find((a) => a.serviceOrderId === orderId && a.isActive);
        if (!isCallerTechnician || !callerTechId || !activeAss || activeAss.technicianId !== callerTechId) {
          return errorResponse(403, 'forbidden', 'Solo el tecnico asignado puede registrar diagnosticos.');
        }
        const { finding, componentToRepair } = body || {};
        const assignment = activeAss;
        const newDiagnostic: Diagnostic = {
          id: crypto.randomUUID ? crypto.randomUUID() : 'd-' + Date.now(),
          serviceOrderId: orderId,
          technicianId: assignment.technicianId,
          finding,
          componentToRepair,
          createdAt: new Date().toISOString(),
        };
        db.diagnostics.push(newDiagnostic);
        saveDB(db);
        return jsonResponse(newDiagnostic, 201);
      }
    }

    // 12. Intervention
    const interventionMatch = pathname.match(/^\/api\/service-order\/([^/]+)\/intervention$/);
    if (interventionMatch) {
      const orderId = interventionMatch[1];
      if (method === 'GET') {
        const list = db.interventions
          .filter((i) => i.serviceOrderId === orderId)
          .map((item) => {
            const war = db.warranties.find((w) => w.interventionId === item.id);
            return {
              ...item,
              warranty: war
                ? {
                    id: war.id,
                    valid: war.valid,
                    kind: war.kind,
                    coverageMonthCount: war.coverageMonthCount,
                  }
                : null,
            };
          });
        return jsonResponse(list);
      }
      if (method === 'POST') {
        const order = db.orders.find((o) => o.id === orderId);
        if (order?.status === 'DELIVERED') {
          return errorResponse(409, 'conflict', 'No se puede registrar intervencion en una orden entregada.');
        }
        const activeAss = db.assignments.find((a) => a.serviceOrderId === orderId && a.isActive);
        if (!isCallerTechnician || !callerTechId || !activeAss || activeAss.technicianId !== callerTechId) {
          return errorResponse(403, 'forbidden', 'Solo el tecnico asignado puede registrar intervenciones.');
        }
        const { description, laborHourCount, part } = body || {};
        const assignment = activeAss;
        const newIntervention: Intervention = {
          id: crypto.randomUUID ? crypto.randomUUID() : 'i-' + Date.now(),
          serviceOrderId: orderId,
          technicianId: assignment?.technicianId || 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1',
          description,
          laborHourCount: Number(laborHourCount) || 1,
          performedAt: new Date().toISOString(),
          part: Array.isArray(part) ? (part as PartUsage[]) : [],
        };
        db.interventions.push(newIntervention);
        saveDB(db);
        return jsonResponse(newIntervention, 201);
      }
    }

    // 13. Warranties (Administrator only)
    if (pathname === '/api/warranty') {
      if (isCallerTechnician) {
        return errorResponse(403, 'forbidden', 'No tiene permiso para acceder a garantias.');
      }
      if (method === 'GET') {
        return jsonResponse(db.warranties);
      }
      if (method === 'POST') {
        const { interventionId, kind, coverageMonthCount } = body || {};
        const months = Number(coverageMonthCount) || 3;
        const now = new Date();
        const exp = new Date(now);
        exp.setMonth(exp.getMonth() + months);

        const interv = db.interventions.find((i) => i.id === interventionId);
        const order = db.orders.find((o) => o.id === interv?.serviceOrderId);

        const newWarranty: Warranty = {
          id: crypto.randomUUID ? crypto.randomUUID() : 'w-' + Date.now(),
          interventionId,
          orderNumber: order?.orderNumber || 'OS-0001',
          vehiclePlate: order?.vehiclePlate || 'ABC123',
          kind: (kind as WarrantyKind) || 'LABOR',
          coverageMonthCount: months,
          issuedAt: now.toISOString(),
          expirationDate: exp.toISOString(),
          valid: true,
        };
        db.warranties.unshift(newWarranty);
        saveDB(db);
        return jsonResponse(newWarranty, 201);
      }
    }

    // 14. Dashboard
    if (pathname === '/api/dashboard' && method === 'GET') {
      const openOrders = db.orders.filter((o) => o.status !== 'DELIVERED');
      const allStatuses: ServiceOrderStatus[] = [
        'RECEIVED',
        'IN_DIAGNOSIS',
        'IN_REPAIR',
        'READY',
        'DELIVERED',
      ];
      const statusCount = allStatuses.map((st) => ({
        status: st,
        count: db.orders.filter((o) => o.status === st).length,
      }));

      const busyTechnicians = db.technicians
        .filter((t) =>
          db.assignments.some((a) => {
            if (a.technicianId !== t.id || !a.isActive) return false;
            const ord = db.orders.find((o) => o.id === a.serviceOrderId);
            return ord && ord.status !== 'DELIVERED';
          }),
        )
        .map((t) => {
          const a = db.assignments.find((asg) => {
            if (asg.technicianId !== t.id || !asg.isActive) return false;
            const ord = db.orders.find((o) => o.id === asg.serviceOrderId);
            return ord && ord.status !== 'DELIVERED';
          });
          const ord = db.orders.find((o) => o.id === a?.serviceOrderId);
          return {
            ...t,
            busy: true,
            activeOrderId: ord?.id || '',
            activeOrderNumber: ord?.orderNumber || '',
            activeVehiclePlate: ord?.vehiclePlate || '',
          };
        });

      const dashboard: Dashboard = {
        openOrderCount: openOrders.length,
        statusCount,
        busyTechnician: busyTechnicians,
      };
      return jsonResponse(dashboard);
    }

    return originalFetch(input, init);
  };
}
