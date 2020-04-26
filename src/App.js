import React from 'react';
import { HashRouter } from 'react-router-dom';

import { Routes } from './routes';
import { Header } from './components/Header';

export function App() {
  return (
    <HashRouter basename='/'>
      <div>test</div>
      <Header />
      <Routes />
    </HashRouter>
  );
}
