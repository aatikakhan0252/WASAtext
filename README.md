# WASAText - Web and Software Architecture
This project is part of the Web and Software Architecture (WASA) course exam at **Sapienza University of Rome**.
## Project Structure
The repository is organized following the standard Go project layout:
- **`cmd/`**: entry points for the application binaries.
  - `webapi/`: Main API server daemon.
  - `healthcheck/`: Server health check tool.
- **`service/`**: Core application logic and libraries.
  - `api/`: API implementation.
  - `database/`: Database access.
  - `globaltime/`: Time wrapper for testing.
- **`webui/`**: Single Page Application (SPA) frontend in Vue.js.
  - Includes Bootstrap dashboard template and Feather icons.
- **`doc/`**: Documentation and OpenAPI specification (`openapi.yaml`).
- **`demo/`**: Configuration files for demonstration.
- **`vendor/`**: Vendored Go dependencies.
### Development Utilities
- **`open-node.sh`**: Helper script to launch a Docker container (`node:20`) for safe frontend development.
