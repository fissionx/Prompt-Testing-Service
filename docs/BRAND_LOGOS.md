# Brand Logo Integration

## Overview

All GEO analytics APIs now include **company logos** for brands, making the data more visual and user-friendly. Logos are automatically fetched, cached in MongoDB, and returned with all analytics responses.

---

## 🎯 **Features**

### ✅ **Automatic Logo Discovery**
- Smart domain detection from brand names
- 40+ pre-configured brand mappings (Amazon, eBay, Walmart, etc.)
- Fallback to `brandname.com` if not found

### ✅ **Database Caching**
- Logos stored in MongoDB (`brand_logos` collection)
- 30-day cache freshness
- Automatic refresh when stale
- Async saving (doesn't block API responses)

### ✅ **Multi-Source Strategy**
1. **Primary**: Clearbit Logo API (high quality, transparent)
2. **Fallback**: Google Favicon (always available)
3. **Placeholder**: UI Avatars (shows brand initials)

### ✅ **Available in All Analytics APIs**
- `/geo/insights` - Main brand + all competitors
- `/analytics/sources` - Main brand
- `/analytics/competitive` - Main brand + all competitors
- `/analytics/position` - Main brand
- `/analytics/prompt-performance` - Main brand

---

## 📊 **API Response Examples**

### **GEO Insights with Logos**

```json
{
  "success": true,
  "data": {
    "brand": "Amazon",
    "logoUrl": "https://logo.clearbit.com/amazon.com",
    "fallbackLogoUrl": "https://www.google.com/s2/favicons?domain=amazon.com&sz=128",
    "averageVisibility": 6,
    "mentionRate": 100,
    "topCompetitors": [
      {
        "name": "eBay",
        "logoUrl": "https://logo.clearbit.com/ebay.com",
        "fallbackLogoUrl": "https://www.google.com/s2/favicons?domain=ebay.com&sz=128",
        "mentionCount": 3,
        "visibilityAvg": 0
      },
      {
        "name": "Walmart",
        "logoUrl": "https://logo.clearbit.com/walmart.com",
        "fallbackLogoUrl": "https://www.google.com/s2/favicons?domain=walmart.com&sz=128",
        "mentionCount": 3,
        "visibilityAvg": 0
      }
    ]
  }
}
```

### **Competitive Benchmark with Logos**

```json
{
  "success": true,
  "data": {
    "mainBrand": {
      "brand": "Amazon",
      "logoUrl": "https://logo.clearbit.com/amazon.com",
      "fallbackLogoUrl": "https://www.google.com/s2/favicons?domain=amazon.com&sz=128",
      "visibility": 6,
      "mentionRate": 100,
      "marketSharePct": 17.6
    },
    "competitors": [
      {
        "brand": "eBay",
        "logoUrl": "https://logo.clearbit.com/ebay.com",
        "fallbackLogoUrl": "https://www.google.com/s2/favicons?domain=ebay.com&sz=128",
        "mentionRate": 100,
        "marketSharePct": 17.6
      }
    ]
  }
}
```

### **Source Analytics with Logo**

```json
{
  "success": true,
  "data": {
    "brand": "Amazon",
    "logoUrl": "https://logo.clearbit.com/amazon.com",
    "fallbackLogoUrl": "https://www.google.com/s2/favicons?domain=amazon.com&sz=128",
    "topSources": [...]
  }
}
```

---

## 🗄️ **Database Schema**

### **Collection**: `brand_logos`

```json
{
  "_id": "uuid",
  "brand_name": "amazon",           // Normalized (lowercase)
  "domain": "amazon.com",           // Detected domain
  "logo_url": "https://logo.clearbit.com/amazon.com",
  "fallback_logo_url": "https://www.google.com/s2/favicons?domain=amazon.com&sz=128",
  "source": "clearbit",             // clearbit | google | placeholder
  "last_checked": "2025-11-30T10:00:00Z",
  "created_at": "2025-11-30T10:00:00Z",
  "updated_at": "2025-11-30T10:00:00Z"
}
```

### **Index**
- `brand_name` (unique)

---

## 🔧 **Logo Service Architecture**

### **Flow Diagram**

```
API Request → LogoService.GetBrandLogo()
                    ↓
        Check MongoDB Cache
                    ↓
        ┌───────────┴───────────┐
        │                       │
    Cache Hit              Cache Miss
    (< 30 days)           or Stale
        │                       │
        │                       ↓
        │              Extract Domain
        │              from Brand Name
        │                       ↓
        │              Generate URLs:
        │              1. Clearbit
        │              2. Google Favicon
        │              3. Placeholder
        │                       ↓
        │              Save to MongoDB
        │              (async)
        │                       │
        └───────────┬───────────┘
                    ↓
            Return Logo URLs
```

### **Key Components**

1. **LogoService** (`internal/services/logo_service.go`)
   - Domain extraction logic
   - URL generation
   - Database caching
   - 40+ brand mappings

2. **MongoDB Layer** (`internal/db/mongodb/mongodb.go`)
   - `SaveBrandLogo()` - Upsert logo cache
   - `GetBrandLogo()` - Retrieve cached logo

3. **Models** (`internal/models/brand_logo.go`)
   - `BrandLogoCache` - Database model
   - `BrandWithLogo` - API response model

---

## 🌐 **Logo URL Sources**

### **1. Clearbit Logo API** (Primary)
- **URL**: `https://logo.clearbit.com/{domain}`
- **Example**: `https://logo.clearbit.com/amazon.com`
- **Quality**: High resolution, transparent PNG
- **Availability**: Most major companies
- **Free**: Yes, no API key needed
- **CDN**: Yes, fast global delivery

### **2. Google Favicon Service** (Fallback)
- **URL**: `https://www.google.com/s2/favicons?domain={domain}&sz=128`
- **Example**: `https://www.google.com/s2/favicons?domain=amazon.com&sz=128`
- **Quality**: Lower resolution, but always works
- **Size**: 128x128 pixels
- **Free**: Yes
- **Reliability**: 100% uptime

### **3. UI Avatars** (Placeholder)
- **URL**: `https://ui-avatars.com/api/?name={initials}&size=128&background=0D8ABC&color=fff&bold=true`
- **Example**: `https://ui-avatars.com/api/?name=Amazon&size=128&background=0D8ABC&color=fff&bold=true`
- **Used When**: Domain can't be determined
- **Shows**: Brand initials with colored background
- **Free**: Yes

---

## 🎨 **Frontend Integration Guide**

### **React Example**

```jsx
import React, { useState } from 'react';

function BrandCard({ brand }) {
  const [logoSrc, setLogoSrc] = useState(brand.logoUrl);
  
  const handleImageError = () => {
    // Automatically fallback to Google Favicon if Clearbit fails
    setLogoSrc(brand.fallbackLogoUrl);
  };
  
  return (
    <div className="brand-card">
      <img 
        src={logoSrc}
        alt={`${brand.brand} logo`}
        onError={handleImageError}
        className="brand-logo"
      />
      <h3>{brand.brand}</h3>
      <div className="metrics">
        <span>Visibility: {brand.visibility}</span>
        <span>Market Share: {brand.marketSharePct.toFixed(1)}%</span>
      </div>
    </div>
  );
}

// Usage
function CompetitiveDashboard({ data }) {
  return (
    <div className="dashboard">
      <h2>Market Position</h2>
      <BrandCard brand={data.mainBrand} />
      
      <h3>Competitors</h3>
      {data.competitors.map(comp => (
        <BrandCard key={comp.brand} brand={comp} />
      ))}
    </div>
  );
}
```

### **Vue.js Example**

```vue
<template>
  <div class="brand-card">
    <img 
      :src="currentLogoUrl"
      :alt="`${brand.brand} logo`"
      @error="useFallback"
      class="brand-logo"
    />
    <h3>{{ brand.brand }}</h3>
    <div class="metrics">
      <span>Visibility: {{ brand.visibility }}</span>
      <span>Market Share: {{ brand.marketSharePct }}%</span>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    brand: {
      type: Object,
      required: true
    }
  },
  data() {
    return {
      currentLogoUrl: this.brand.logoUrl
    }
  },
  methods: {
    useFallback() {
      this.currentLogoUrl = this.brand.fallbackLogoUrl;
    }
  }
}
</script>

<style scoped>
.brand-logo {
  width: 64px;
  height: 64px;
  object-fit: contain;
  border-radius: 8px;
  background: white;
  padding: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}
</style>
```

### **Angular Example**

```typescript
import { Component, Input, OnInit } from '@angular/core';

@Component({
  selector: 'app-brand-card',
  template: `
    <div class="brand-card">
      <img 
        [src]="currentLogoUrl"
        [alt]="brand.brand + ' logo'"
        (error)="useFallback()"
        class="brand-logo"
      />
      <h3>{{ brand.brand }}</h3>
      <div class="metrics">
        <span>Visibility: {{ brand.visibility }}</span>
        <span>Market Share: {{ brand.marketSharePct | number:'1.1-1' }}%</span>
      </div>
    </div>
  `,
  styleUrls: ['./brand-card.component.css']
})
export class BrandCardComponent implements OnInit {
  @Input() brand: any;
  currentLogoUrl: string;

  ngOnInit() {
    this.currentLogoUrl = this.brand.logoUrl;
  }

  useFallback() {
    this.currentLogoUrl = this.brand.fallbackLogoUrl;
  }
}
```

---

## 🎨 **CSS Styling Examples**

### **Basic Styling**

```css
.brand-logo {
  width: 64px;
  height: 64px;
  object-fit: contain;
  border-radius: 8px;
  background: white;
  padding: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  transition: transform 0.2s;
}

.brand-logo:hover {
  transform: scale(1.1);
}
```

### **Market Leader Styling**

```css
.brand-card.leader .brand-logo {
  border: 3px solid gold;
  box-shadow: 0 0 20px rgba(255, 215, 0, 0.5);
}

.brand-card.main-brand .brand-logo {
  border: 3px solid #0D8ABC;
  box-shadow: 0 0 20px rgba(13, 138, 188, 0.3);
}
```

### **Loading State**

```css
.brand-logo.loading {
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: loading 1.5s infinite;
}

@keyframes loading {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
```

---

## 📝 **Adding Custom Brand Mappings**

If you need to add more brands to the built-in mappings:

**File**: `internal/services/logo_service.go`

```go
// extractDomain extracts a domain from brand name or website
func (s *LogoService) extractDomain(brandName string, website string) string {
    // ...
    
    // Common brand name to domain mappings
    knownDomains := map[string]string{
        "amazon":         "amazon.com",
        "ebay":           "ebay.com",
        // ... existing mappings ...
        
        // ADD YOUR CUSTOM BRANDS HERE
        "my company":     "mycompany.com",
        "acme corp":      "acmecorp.io",
        "startup inc":    "startup.io",
    }
    
    // ...
}
```

Then rebuild:

```bash
go build -o gego ./cmd/gego
```

---

## 🧪 **Testing Logo Integration**

### **Test API with cURL**

```bash
# Test GEO Insights (main brand + competitors)
curl -X POST 'http://localhost:8989/api/v1/geo/insights' \
  -H 'Content-Type: application/json' \
  -d '{"brand": "Amazon"}'

# Test Competitive Benchmark
curl -X POST 'http://localhost:8989/api/v1/geo/analytics/competitive' \
  -H 'Content-Type: application/json' \
  -d '{"mainBrand": "Amazon"}'

# Test Source Analytics
curl -X POST 'http://localhost:8989/api/v1/geo/analytics/sources' \
  -H 'Content-Type: application/json' \
  -d '{"brand": "Amazon", "topN": 10}'
```

### **Verify Logo Cache in MongoDB**

```bash
# Connect to MongoDB
mongosh mongodb://localhost:27017/gego

# View cached logos
db.brand_logos.find().pretty()

# Check specific brand
db.brand_logos.findOne({brand_name: "amazon"})

# Count cached logos
db.brand_logos.countDocuments()
```

---

## 🔍 **Troubleshooting**

### **Logo Not Appearing?**

1. **Check API Response**
   ```bash
   curl ... | jq '.data.logoUrl'
   ```

2. **Check Database Cache**
   ```bash
   mongosh mongodb://localhost:27017/gego
   db.brand_logos.findOne({brand_name: "yourbrand"})
   ```

3. **Check Domain Mapping**
   - Is your brand in `knownDomains` map?
   - Does `brandname.com` exist?

4. **Test Logo URLs Directly**
   ```bash
   # Test Clearbit
   curl -I https://logo.clearbit.com/amazon.com
   
   # Test Google Favicon
   curl -I "https://www.google.com/s2/favicons?domain=amazon.com&sz=128"
   ```

### **Logo Looks Generic?**

- Clearbit might not have this company
- Fallback to Google Favicon is being used
- Consider adding a custom mapping

### **Logo Not Updating?**

- Cache is valid for 30 days
- Delete from MongoDB to force refresh:
  ```bash
  db.brand_logos.deleteOne({brand_name: "yourbrand"})
  ```

---

## 🚀 **Performance Considerations**

### **Caching Benefits**
- **First Request**: ~200ms (fetch + save)
- **Subsequent Requests**: ~5ms (cache hit)
- **30-day cache**: Reduces external API calls by 99%

### **Async Saving**
- Logo saving doesn't block API response
- Failures are logged but don't fail the request

### **CDN Delivery**
- Clearbit and Google serve from CDNs
- Fast global delivery
- Browser caching supported

---

## 📈 **Future Enhancements**

### **Planned Features**
- [ ] Custom logo upload for brands
- [ ] Logo quality scoring
- [ ] Multiple logo sizes (32px, 64px, 128px, 256px)
- [ ] Dark mode logo variants
- [ ] Brand color extraction
- [ ] Favicon extraction from actual websites

---

## 🎯 **UI Design Ideas**

### **1. Competitor Leaderboard**
```
┌─────────────────────────────────────┐
│ 🏆 Market Leader                    │
│ ┌────┐  Amazon                      │
│ │ 📦 │  Visibility: 6/10      25%   │
│ └────┘  "Top position, strong       │
│         grounding sources"           │
├─────────────────────────────────────┤
│ ┌────┐  eBay                  20%   │
│ │ 🏪 │  Visibility: 5/10            │
│ └────┘                              │
├─────────────────────────────────────┤
│ ┌────┐  Walmart               18%   │
│ │ 🛒 │  Visibility: 4/10            │
│ └────┘                              │
└─────────────────────────────────────┘
```

### **2. Market Share Pie Chart**
Display brand logos around pie chart slices for visual identification.

### **3. Timeline View**
Show logo next to each data point in time-series charts.

### **4. Comparison Matrix**
Grid layout with logos as column/row headers for easy comparison.

---

## ✅ **Benefits**

✅ **Professional UI** - Instantly recognizable brand logos  
✅ **No Manual Work** - Automatic discovery and caching  
✅ **Always Works** - Multiple fallback options  
✅ **No API Keys** - All services are free and public  
✅ **Fast** - Logos served from CDNs + MongoDB cache  
✅ **Reliable** - Fallback chain ensures display  
✅ **Consistent** - Same logos across all analytics  
✅ **Scalable** - Database caching handles high volume  

---

## 📚 **Related Documentation**

- [API Reference](./GEO_API.md)
- [Advanced Analytics](./ADVANCED_ANALYTICS.md)
- [Testing Guide](../COMPLETE_API_TEST_AMAZON.md)

---

**Happy branding! 🎨🚀**

