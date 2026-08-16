# Go-React-Fleet-MVP

## Running the Application

### 1. Backend (Go)
To start the backend, from the root of this repository, run:
```bash
cd backend
DATA_DIR=../drive-download-20260816T202653Z-1-001/data go run main.go
```
The API will start on `http://localhost:8080`.

### 2. Frontend (React)
To start the frontend, open a new terminal and run:
```bash
cd frontend
npm install
npm run dev
```
The frontend will start on `http://localhost:5173` (or the next available port).

---

### Running the Data Simulator
If you need to re-run the live simulator to push data to the ingest endpoint, run:
```bash
cd drive-download-20260816T202653Z-1-001
python3 tools/simulate_live.py --all --speed max --url http://localhost:8080/api/v1/ingest/telemetry --token nxc_demo_token_9f2b41c7
```
