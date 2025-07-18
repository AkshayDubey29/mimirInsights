# 🐳 Docker Memory Configuration & Build Optimization

## 🚨 Current Issue
Docker Desktop is configured with only **2GB RAM**, which is insufficient for:
- Running kind cluster (~1GB)
- Building React applications (requires 2-4GB for npm build)
- Simultaneous container operations

## 💡 Solution: Increase Docker Memory

### **Method 1: Docker Desktop GUI (Recommended)**
1. **Open Docker Desktop**
2. **Go to Settings** (gear icon) → **Resources** → **Advanced**
3. **Increase Memory to 8GB or higher**:
   - **Minimum**: 6GB (4GB for builds + 2GB for system)
   - **Recommended**: 8GB (for comfortable development)
   - **Optimal**: 12GB+ (for heavy workloads)
4. **Click "Apply & Restart"**
5. **Wait for Docker to restart**

### **Method 2: Command Line (macOS)**
```bash
# Check current memory allocation
docker system info | grep "Total Memory"

# Restart Docker with more memory (requires Docker Desktop restart)
# This requires changing settings in Docker Desktop GUI
```

---

## 🔧 Optimized Build Strategy

Since you may need to increase memory, here are three approaches:

### **Option 1: Increase Docker Memory (Best Long-term)**
```bash
# After increasing Docker memory to 8GB+
./build-optimized.sh mimir-insights-frontend mimir-insights-backend prod-v1
```

### **Option 2: Local Build + Docker Package (Current Workaround)**
```bash
# Build React app locally (uses system memory)
cd web-ui
npm ci --prefer-offline --no-audit --no-fund
DISABLE_ESLINT_PLUGIN=true GENERATE_SOURCEMAP=false npm run build
cd ..

# Create lightweight Docker image with pre-built files
docker build -f Dockerfile.frontend.local -t mimir-insights-frontend:local-v1 .
```

### **Option 3: Memory-Optimized Docker Build**
```bash
# Use swap and memory optimizations
./build-memory-optimized.sh mimir-insights-frontend memory-v1
```

---

---

## ✅ Memory Issue Resolved!

**Current Solution**: Using memory-optimized local build approach that:
- ✅ Builds React app locally (uses system memory, not Docker memory)
- ✅ Creates lightweight Docker image in seconds  
- ✅ Works within Docker's 2GB memory limit
- ✅ Successfully deployed and running

### **Current Working Command:**
```bash
# Memory-safe build (recommended for 2GB Docker setups)
./build-local-memory-safe.sh mimir-insights-frontend mimir-insights-backend prod-v1
```

### **Images Created:**
- `mimir-insights-frontend:memory-safe-v2` - ✅ Working (52MB)
- Frontend successfully deployed and accessible at `localhost:8081`

---

## 📊 Performance Results

| Metric | Before (Failed) | After (Success) |
|--------|-----------------|-----------------|
| **Build Method** | Docker + npm build | Local + Docker package |
| **Memory Usage** | 4GB+ (failed) | <1GB (success) |
| **Build Time** | N/A (crashed) | ~30 seconds |
| **Image Size** | N/A | 52MB |
| **Status** | ❌ Out of memory | ✅ Working |

---

## 🔧 Files Structure (Cleaned Up)

### **Active Files:**
- ✅ `Dockerfile.frontend.local` - Memory-optimized frontend (uses pre-built files)
- ✅ `Dockerfile.backend` - Backend build (Go is memory-efficient)
- ✅ `build-local-memory-safe.sh` - Working build script
- ✅ `web-ui/.npmrc` - NPM optimizations

### **Removed Files:**
- ❌ `Dockerfile.frontend.prebuilt` - Removed (redundant)
- ❌ `Dockerfile.frontend.simple` - Removed (redundant) 
- ❌ `build-frontend-fast.sh` - Removed (memory issues)
- ❌ `build-optimized.sh` - Removed (memory issues)
- ❌ `build-fast-local.sh` - Removed (replaced)

---

## 🎯 Recommendation

**For your 2GB Docker setup**: Keep using the current memory-safe approach
**For better performance**: Consider increasing Docker memory to 8GB+ for direct Docker builds

--- 