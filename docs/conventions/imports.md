# Import Ordering

### 3.1 Five Groups

Imports are organized into five groups, separated by blank lines:

```go
import (
    // 1. Standard library
    "context"
    "database/sql"
    "fmt"
    "time"

    // 2. bin-common-handler (shared monorepo library)
    commonaddress "monorepo/bin-common-handler/models/address"
    commonidentity "monorepo/bin-common-handler/models/identity"
    "monorepo/bin-common-handler/pkg/notifyhandler"
    "monorepo/bin-common-handler/pkg/requesthandler"
    "monorepo/bin-common-handler/pkg/utilhandler"

    // 3. Cross-service models (other bin-* services)
    cucustomer "monorepo/bin-customer-manager/models/customer"
    fmaction "monorepo/bin-flow-manager/models/action"

    // 4. Third-party packages
    "github.com/Masterminds/squirrel"
    "github.com/gofrs/uuid"
    "github.com/pkg/errors"
    "github.com/sirupsen/logrus"
    gomock "go.uber.org/mock/gomock"

    // 5. Local service packages (same service)
    "monorepo/bin-agent-manager/models/agent"
    "monorepo/bin-agent-manager/pkg/cachehandler"
    "monorepo/bin-agent-manager/pkg/dbhandler"
)
```

**Rationale:** Consistent grouping makes import blocks scannable and prevents merge conflicts.

**Wrong:**
```go
// WRONG — all mixed together
import (
    "context"
    "monorepo/bin-agent-manager/models/agent"
    "github.com/gofrs/uuid"
    "monorepo/bin-common-handler/pkg/requesthandler"
    "fmt"
)

---
