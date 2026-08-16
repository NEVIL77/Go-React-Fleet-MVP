import { BrowserRouter, Route, Routes } from "react-router-dom";
import { TripDetailPage } from "./pages/TripDetailPage";
import { TripsPage } from "./pages/TripsPage";
import { isMockMode } from "./api/client";
import "./App.css";

export default function App() {
  return (
    <BrowserRouter>
      <div className="app">
        <header>
          <strong>Fleet Safety</strong>
          {isMockMode() && <span className="mock-badge">mock data</span>}
        </header>
        <main>
          <Routes>
            <Route path="/" element={<TripsPage />} />
            <Route path="/trips/:tripId" element={<TripDetailPage />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
