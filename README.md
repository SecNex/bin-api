# SecNex Bin API 🗑️

A minimalist HTTP API for testing and debugging HTTP requests during development. This "bin" API receives arbitrary HTTP requests and returns all request details as structured JSON responses.

## What is a Bin API?

A Bin API (also called Request Bin or HTTP Bin) is a development tool that accepts HTTP requests and returns their details. It works like a "digital trash bin" for HTTP requests - everything sent to it gets analyzed and returned in a structured format.

## Functionality

The SecNex Bin API offers the following features:

### 🔍 **Request Analysis**

- **Body Parsing**: Processes JSON bodies and returns them structured
- **Header Extraction**: Collects all HTTP headers and makes them searchable
- **Query Parameters**: Extracts and formats URL parameters
- **HTTP Methods**: Supports all standard HTTP methods (GET, POST, PUT, DELETE, etc.)

### 📊 **Structured Responses**

Every request is returned as a JSON object:

```json
{
  "body": {...},      // Request body as JSON
  "headers": {...},   // All HTTP headers
  "queries": {...},   // URL query parameters
  "params": {}        // URL parameters (reserved for future extensions)
}
```

### 🔧 **Logging & Monitoring**

- **Nginx-style Logging**: Detailed request logs in standard format
- **Response Time Tracking**: Measures and logs response times
- **Health Check**: `/healthz` endpoint for container orchestration

## Benefits for Development

### 🚀 **API Development & Testing**

- **Webhook Testing**: Perfect for testing webhooks without real target APIs
- **Request Debugging**: Analyze what your application actually sends
- **Integration Tests**: Use as mock endpoint in your tests
- **Payload Validation**: Verify JSON structure and data format

### 🛠️ **DevOps & CI/CD**

- **Container-Ready**: Runs out-of-the-box in Docker/Kubernetes
- **Health Checks**: Built-in health monitoring
- **Lightweight**: Minimal resource consumption (~10MB container)
- **Zero-Config**: No configuration required

### 🔬 **Debugging Scenarios**

- **CORS Issues**: Analyze cross-origin request headers
- **Content-Type Issues**: Verify header configuration
- **Authentication Debugging**: Test authorization headers
- **Rate Limiting**: Simulate API responses for load tests

### 💡 **Developer Productivity**

- **Instant Feedback**: Immediate response to all requests
- **No Registration**: Local operation without external dependencies
- **Version Control**: Deterministic behavior for reproducible tests
- **Multi-Environment**: Easy deployment in Dev/Stage/Prod

## Installation & Usage

### 🐳 **Docker (Recommended)**

```bash
# Start container
docker run --name bin-api -p 8081:8081 -d ghcr.io/secnex/bin-api:latest

# Health check
curl http://localhost:8081/healthz

# Test request
curl -X POST http://localhost:8081/api/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -d '{"user": "alice", "action": "login"}'
```

### 🔧 **Docker Compose**

```yaml
version: "3.8"
services:
  bin-api:
    image: ghcr.io/secnex/bin-api:latest
    ports:
      - "8081:8081"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### 💻 **Local Development**

```bash
# Clone repository
git clone https://github.com/secnex/bin-api.git
cd bin-api

# Install dependencies
go mod download

# Start server
go run main.go

# Alternative: Build binary
go build -o bin-api main.go
./bin-api
```

# Deploy and Host

## About Hosting

The SecNex Bin API is designed for easy deployment across various hosting platforms and environments. Its lightweight Go-based architecture and containerized design make it ideal for both cloud and on-premises deployments.

### **Hosting Characteristics**

- **Resource Efficient**: Minimal CPU and memory footprint
- **Stateless Design**: No database dependencies or persistent storage
- **Container-Native**: Optimized for modern container orchestration
- **Multi-Platform**: Supports x86_64 and ARM architectures
- **High Availability**: Built-in health checks for load balancer integration

## Why Deploy

### **Production Benefits**

- **Team Collaboration**: Shared testing endpoint for distributed teams
- **CI/CD Integration**: Consistent testing environment across pipelines
- **Security**: Keep sensitive request data within your infrastructure
- **Performance**: Reduced latency compared to external services
- **Reliability**: No dependency on third-party service availability

### **Cost Efficiency**

- **No External Fees**: Eliminate subscription costs for request bin services
- **Minimal Infrastructure**: Single container deployment
- **Auto-Scaling**: Container orchestration handles traffic spikes
- **Resource Optimization**: Efficient Go runtime with small memory footprint

### **Compliance & Security**

- **Data Privacy**: All requests stay within your network
- **GDPR Compliance**: No external data transmission
- **Network Isolation**: Deploy in private subnets or VPCs
- **Custom Security**: Implement your own authentication if needed

## Common Use Cases

### **Development Teams**

- **Microservices Testing**: Mock external APIs during development
- **Webhook Development**: Test webhook integrations without external dependencies
- **API Gateway Testing**: Validate request transformations and routing
- **Load Testing**: Simulate high-volume API endpoints

### **DevOps & Infrastructure**

- **CI/CD Pipeline Testing**: Validate deployment scripts and automation
- **Monitoring Setup**: Test alerting webhooks and notification systems
- **Container Orchestration**: Practice deployment strategies in Kubernetes
- **Network Testing**: Validate service mesh and ingress configurations

### **Security & Compliance**

- **Penetration Testing**: Analyze request patterns in controlled environment
- **API Security Audits**: Validate header handling and input processing
- **Compliance Testing**: Ensure request logging meets regulatory requirements
- **Incident Response**: Simulate security events for team training

### **Educational & Training**

- **API Development Training**: Teach HTTP concepts with real examples
- **Debugging Workshops**: Practice troubleshooting HTTP issues
- **Integration Testing**: Demonstrate API integration patterns
- **Performance Analysis**: Understand request/response characteristics

## Dependencies for

### Deployment Dependencies

#### **Container Runtime**

- **Docker**: Version 20.10+ (recommended)
- **Podman**: Version 3.0+ (alternative)
- **containerd**: Version 1.5+ (Kubernetes environments)

#### **Orchestration Platforms**

- **Kubernetes**: Version 1.20+

  ```yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: bin-api
  spec:
    replicas: 2
    selector:
      matchLabels:
        app: bin-api
    template:
      metadata:
        labels:
          app: bin-api
      spec:
        containers:
          - name: bin-api
            image: ghcr.io/secnex/bin-api:latest
            ports:
              - containerPort: 8081
            livenessProbe:
              httpGet:
                path: /healthz
                port: 8081
              initialDelaySeconds: 5
              periodSeconds: 10
  ```

- **Docker Swarm**: Version 20.10+
  ```bash
  docker service create \
    --name bin-api \
    --replicas 3 \
    --publish 8081:8081 \
    --health-cmd "curl -f http://localhost:8081/healthz || exit 1" \
    --health-interval 30s \
    ghcr.io/secnex/bin-api:latest
  ```

#### **Cloud Platforms**

- **AWS**: ECS, EKS, Fargate, Lambda (with custom runtime)
- **Google Cloud**: GKE, Cloud Run, Compute Engine
- **Azure**: AKS, Container Instances, App Service
- **DigitalOcean**: Kubernetes, App Platform, Droplets

#### **Infrastructure as Code**

- **Terraform**: Version 1.0+

  ```hcl
  resource "docker_container" "bin_api" {
    name  = "bin-api"
    image = "ghcr.io/secnex/bin-api:latest"

    ports {
      internal = 8081
      external = 8081
    }

    healthcheck {
      test         = ["CMD", "curl", "-f", "http://localhost:8081/healthz"]
      interval     = "30s"
      timeout      = "10s"
      retries      = 3
      start_period = "5s"
    }
  }
  ```

- **Ansible**: Version 2.9+
- **Helm**: Version 3.0+ (for Kubernetes deployments)

#### **Monitoring & Observability**

- **Prometheus**: For metrics collection
- **Grafana**: For visualization dashboards
- **ELK Stack**: For log aggregation and analysis
- **Jaeger/Zipkin**: For distributed tracing

#### **Load Balancing**

- **Nginx**: Version 1.18+
- **HAProxy**: Version 2.0+
- **Traefik**: Version 2.0+
- **Cloud Load Balancers**: AWS ALB/ELB, GCP Load Balancer, Azure Load Balancer

#### **Minimum System Requirements**

- **CPU**: 0.1 vCPU (100m in Kubernetes)
- **Memory**: 32MB RAM (64MB recommended)
- **Storage**: 50MB for container image
- **Network**: HTTP/HTTPS access on port 8081

## Technical Details

- **Language**: Go 1.24.3
- **Framework**: Standard `net/http` library
- **Container**: Alpine Linux (Multi-stage build)
- **Port**: 8081 (default)
- **Protocol**: HTTP/1.1
- **Content-Type**: Supports all formats, JSON parsing for application/json

## Why Choose SecNex Bin API?

✅ **Zero-Configuration** - Starts immediately without setup  
✅ **Production-Ready** - Container-optimized with health checks  
✅ **Developer-Friendly** - Structured, readable output  
✅ **Lightweight** - Minimal resource consumption  
✅ **Open Source** - Transparent and extensible  
✅ **Standards-Compliant** - Follows HTTP specifications

The SecNex Bin API is the perfect tool for modern development teams who want to efficiently develop, test, and debug APIs. Whether for microservices, webhook integration, or API mocking - this solution significantly accelerates your development process.

## License

This project is licensed under an open-source license. See the LICENSE file for more details.

```bash
curl http://localhost:8081/test
```
