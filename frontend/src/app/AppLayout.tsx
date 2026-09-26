import { useState } from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';

import { useSession } from '../shared/SessionContext';
import { ChangePasswordModal } from '../features/login/ChangePasswordModal';

interface NavigationItem {
  to: string;
  label: string;
  administratorOnly: boolean;
}

const NAVIGATION: NavigationItem[] = [
  { to: '/dashboard', label: 'Panel', administratorOnly: false },
  { to: '/service-orders', label: 'Ordenes', administratorOnly: false },
  { to: '/customers', label: 'Clientes', administratorOnly: true },
  { to: '/vehicles', label: 'Vehiculos', administratorOnly: true },
  { to: '/technicians', label: 'Tecnicos', administratorOnly: true },
  { to: '/warranties', label: 'Garantias', administratorOnly: true },
];

export function AppLayout() {
  const { session, signOut, isAdministrator } = useSession();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);

  const leave = () => {
    signOut();
    navigate('/login', { replace: true });
  };

  const closeMenu = () => {
    setMenuOpen(false);
  };

  return (
    <>
      <header className="app-header">
        <div className="app-header__bar">
          <h1 className="app-header__brand">Soporte Tecnico Automotriz</h1>
          <button
            type="button"
            className="app-header__menu-toggle"
            onClick={() => setMenuOpen(!menuOpen)}
            aria-label={menuOpen ? 'Cerrar menu' : 'Abrir menu'}
            aria-expanded={menuOpen}
          >
            {menuOpen ? '✕' : '☰'}
          </button>
        </div>
        <nav
          className={`app-header__nav ${menuOpen ? 'app-header__nav--open' : ''}`}
          aria-label="Navegacion principal"
        >
          {NAVIGATION.filter((item) => isAdministrator || !item.administratorOnly).map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              onClick={closeMenu}
              className={({ isActive }) =>
                'app-header__link' + (isActive ? ' app-header__link--active' : '')
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className={`app-header__user ${menuOpen ? 'app-header__user--open' : ''}`}>
          <span className="app-header__username">{session?.fullName}</span>
          <button type="button" className="button button--secondary app-header__logout" onClick={leave}>
            Salir
          </button>
        </div>
      </header>
      <main className="app-main">
        <Outlet />
      </main>
      <ChangePasswordModal />
    </>
  );
}
