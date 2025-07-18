import React from 'react';

const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <div>
      <nav style={{ padding: '1rem', background: '#132f4c', color: '#fff' }}>
        <span style={{ marginRight: 16 }}>MimirInsights</span>
        <a href="/" style={{ color: '#fff', marginRight: 8 }}>Dashboard</a>
        <a href="/tenants" style={{ color: '#fff', marginRight: 8 }}>Tenants</a>
        <a href="/limits" style={{ color: '#fff', marginRight: 8 }}>Limits</a>
        <a href="/config" style={{ color: '#fff', marginRight: 8 }}>Config</a>
        <a href="/environment-status" style={{ color: '#fff', marginRight: 8 }}>Environment</a>
        <a href="/reports" style={{ color: '#fff' }}>Reports</a>
      </nav>
      <main style={{ padding: '2rem' }}>{children}</main>
    </div>
  );
};

export default Layout; 