export default {
  "home": {
    "mirrors": {
      "title": "Access Mirrors",
      "description": "Choose a nearby route for the same API service",
      "currentSite": "Main Site",
      "default": "Default",
      "action": "Sign in to create and manage keys through this route"
    },
    "customLinks": {
      "title": "Featured Links",
      "description": "Common pages and external resources"
    }
  },
  "nav": {
    "supportTickets": "Support",
    "ticketManagement": "Tickets",
    "apiManagement": "API Management",
    "services": "Services",
    "myInvoices": "My Invoices",
    "invoiceManagement": "Invoice Management"
  },
  "supportTickets": {
    "title": "Support",
    "description": "Submit issues, feedback, or service requests and review replies",
    "create": "New Ticket",
    "view": "View Ticket",
    "detail": "Ticket Detail",
    "reply": "Reply",
    "sendReply": "Send Reply",
    "searchPlaceholder": "Search tickets...",
    "empty": "No tickets",
    "emptyDescription": "Your issues and feedback will appear here after submission.",
    "created": "Ticket created",
    "replied": "Reply sent",
    "failedToLoad": "Failed to load tickets",
    "failedToLoadDetail": "Failed to load ticket detail",
    "failedToCreate": "Failed to create ticket",
    "failedToReply": "Failed to send reply",
    "filters": {
      "allStatus": "All Status",
      "allCategories": "All Categories"
    },
    "columns": {
      "title": "Title",
      "category": "Category",
      "status": "Status",
      "priority": "Priority",
      "lastMessageAt": "Last Message"
    },
    "form": {
      "title": "Title",
      "category": "Category",
      "priority": "Priority",
      "status": "Status",
      "content": "Content"
    },
    "status": {
      "open": "Open",
      "pending": "Pending",
      "resolved": "Resolved",
      "closed": "Closed"
    },
    "priority": {
      "low": "Low",
      "normal": "Normal",
      "high": "High"
    },
    "category": {
      "feedback": "Feedback",
      "bug": "Bug",
      "billing": "Billing",
      "account": "Account",
      "other": "Other"
    },
    "role": {
      "user": "User",
      "admin": "Admin"
    }
  },
  "invoice": {
    "title": "My Invoices",
    "description": "Manage invoice-eligible recharge codes and invoice files",
    "management": "Invoice Management",
    "managementDescription": "Review requests and upload issued invoice files",
    "availableAmount": "Available Amount",
    "pendingAmount": "Pending Amount",
    "issuedAmount": "Issued Amount",
    "availableRecharges": "Available Recharges",
    "createRequest": "Invoice Request",
    "invoiceTitle": "Invoice Title",
    "taxNo": "Tax Number",
    "note": "Note",
    "selectedCount": "Selected Recharges",
    "selectedAmount": "Selected Amount",
    "submitRequest": "Submit Request",
    "history": "Invoice History",
    "id": "ID",
    "amount": "Amount",
    "status": "Status",
    "createdAt": "Requested At",
    "usedAt": "Recharge Time",
    "code": "Code",
    "user": "User",
    "recharges": "Recharges",
    "detail": "Invoice Detail",
    "selectedRecharges": "Selected Recharges",
    "fileName": "File Name",
    "fileSize": "File Size",
    "uploadInvoice": "Upload Invoice",
    "uploadConfirmTitle": "Confirm Invoice Upload",
    "uploadConfirmMessage": "Upload \"{file}\" to invoice #{id}? The request will be marked as issued after upload.",
    "deleteFile": "Delete File",
    "deleteFileConfirmTitle": "Delete Invoice File",
    "deleteFileConfirmMessage": "Delete the uploaded file for invoice #{id}? The request will return to pending status.",
    "deleteFileSuccess": "Invoice file deleted and request returned to pending",
    "viewFile": "View File",
    "preview": "Preview",
    "previewUnavailable": "Preview is not available for this file type. Download it to view.",
    "exportPendingCsv": "Export Pending CSV",
    "exportOlderThanSixHours": "Only export older than 6 hours",
    "exportSuccess": "Pending invoice data exported",
    "settings": "Invoice Settings",
    "minInvoiceAmount": "Minimum Invoice Amount",
    "minInvoiceAmountHint": "Users cannot submit invoice requests below this amount. 0 means no limit.",
    "maxInvoiceAmount": "Maximum Invoice Amount",
    "maxInvoiceAmountHint": "Users cannot submit invoice requests above this amount. 0 means no limit.",
    "invoiceAmountLimitHint": "Users cannot submit invoice requests below the minimum or above the maximum. 0 means no limit.",
    "minInvoiceAmountNotice": "Minimum invoice amount: {amount}. Requests below this amount cannot be submitted.",
    "maxInvoiceAmountNotice": "Maximum invoice amount: {amount}. Requests above this amount cannot be submitted.",
    "belowMinInvoiceAmount": "Selected amount is below the minimum invoice amount {amount}",
    "aboveMaxInvoiceAmount": "Selected amount is above the maximum invoice amount {amount}",
    "saveSettings": "Save Settings",
    "settingsSaved": "Invoice settings saved",
    "download": "Download",
    "withdraw": "Withdraw",
    "withdrawConfirmTitle": "Withdraw Invoice Request",
    "withdrawConfirmMessage": "Withdraw invoice request #{id}? The selected recharges will become invoice-eligible again.",
    "searchPlaceholder": "Search user, title, or tax number",
    "noRecharges": "No invoice-eligible recharge codes yet",
    "noInvoices": "No invoice requests found",
    "createSuccess": "Invoice request submitted",
    "withdrawSuccess": "Invoice request withdrawn",
    "uploadSuccess": "Invoice file uploaded",
    "statuses": {
      "pending": "Pending",
      "issued": "Issued",
      "rejected": "Rejected",
      "withdrawn": "Withdrawn"
    },
    "errors": {
      "INVOICE_RECHARGE_UNAVAILABLE": "The selected recharge is unavailable or already used",
      "INVOICE_FILE_UNAVAILABLE": "Invoice file is unavailable",
      "INVOICE_FILE_TOO_LARGE": "Invoice file is too large",
      "INVOICE_AMOUNT_BELOW_MINIMUM": "Selected amount is below the minimum invoice amount {minimum}",
      "INVOICE_AMOUNT_ABOVE_MAXIMUM": "Selected amount is above the maximum invoice amount {maximum}",
      "INVOICE_WITHDRAW_UNAVAILABLE": "This invoice request cannot be withdrawn",
      "INVOICE_WITHDRAW_EXPIRED": "The 6-hour withdrawal window has expired",
      "INVOICE_UPLOAD_UNAVAILABLE": "This invoice request cannot be uploaded",
      "INVOICE_MIN_AMOUNT_INVALID": "Minimum invoice amount must be greater than or equal to 0",
      "INVOICE_MAX_AMOUNT_INVALID": "Maximum invoice amount must be greater than or equal to 0",
      "INVOICE_AMOUNT_RANGE_INVALID": "Maximum invoice amount must be greater than or equal to the minimum invoice amount",
      "INVOICE_TITLE_REQUIRED": "Invoice title is required",
      "INVOICE_RECHARGES_REQUIRED": "Select at least one recharge",
      "INVOICE_NOT_FOUND": "Invoice request not found"
    }
  },
  "keys": {
    "imageKey": "Image Key",
    "setImageKey": "Use for images",
    "imageKeyCurrent": "Current image key used for {token}",
    "imageKeyIneligible": "Only active, unexpired keys in image-enabled OpenAI groups can be used for images",
    "imageKeySetSuccess": "Image key selected",
    "failedToSetImageKey": "Failed to set image key",
    "ccsImport": {
      "title": "Import to CCS",
      "description": "Select the access mirror to import into CCS.",
      "clientSection": "Client Type",
      "mirrorSection": "Access Mirror",
      "currentSite": "Main Site",
      "defaultMirror": "Default",
      "selectMirror": "Please select an access mirror"
    }
  },
  "admin": {
    "groups": {
      "imagePricing": {
        "responsesImageGenerationRedirect": "Responses image redirect target",
        "responsesImageGenerationRedirectHint": "Explicit /responses requests with the image_generation tool are forwarded to the selected OpenAI image group. Codex image bridging can use this target too. Billing still belongs to the current API key group.",
        "responsesImageGenerationRedirectGroupPlaceholder": "Select OpenAI image group",
        "noOpenAIImageGroupsAvailable": "No OpenAI image-generation groups available"
      }
    },
    "channels": {
      "form": {
        "codexImageGenerationBridgeHint": "When enabled, Codex /responses text requests in OpenAI groups may be automatically given the image_generation tool. This can be paired with the Responses image redirect target on the group."
      }
    },
    "accounts": {
      "openai": {
        "codexCLIOnlyAllowClaudeCode": "Also allow Claude Code's Codex plugin",
        "codexCLIOnlyAllowClaudeCodeDesc": "Only takes effect when the switch above is on. Additionally allows requests from the Claude Code Codex plugin (exact match on originator=Claude Code) without weakening blocking of other non-official clients.",
        "codexImageGenerationBridge": "Codex image-generation bridge",
        "codexImageGenerationBridgeDesc": "Account policy takes precedence over channel and global settings. Only controls whether Codex requests through the /responses text endpoint receive the image_generation tool; it can be paired with the group Responses image redirect target and does not affect standalone image-generation endpoints.",
        "codexImageGenerationBridgeInherit": "Follow channel",
        "codexImageGenerationBridgeInheritDesc": "Do not write an account override; use the channel or global policy.",
        "codexImageGenerationBridgeEnabled": "Force on",
        "codexImageGenerationBridgeEnabledDesc": "Allow image tool injection for Codex /responses requests.",
        "codexImageGenerationBridgeDisabled": "Force off",
        "codexImageGenerationBridgeDisabledDesc": "Block image tool injection for Codex /responses requests.",
        "codexImageGenerationBridgeBadgeInherit": "Channel policy",
        "codexImageGenerationBridgeBadgeEnabled": "Account on",
        "codexImageGenerationBridgeBadgeDisabled": "Account off"
      }
    },
    "supportTickets": {
      "title": "Ticket Management",
      "description": "Review user tickets, reply, and update handling status",
      "create": "Create User Ticket",
      "edit": "Update Ticket",
      "view": "View Ticket",
      "user": "User",
      "userId": "User ID",
      "searchPlaceholder": "Search title, email, or username...",
      "empty": "No tickets",
      "emptyDescription": "User-submitted tickets will appear here.",
      "created": "Ticket created",
      "updated": "Ticket updated",
      "failedToLoad": "Failed to load tickets",
      "failedToLoadDetail": "Failed to load ticket detail",
      "failedToCreate": "Failed to create ticket",
      "failedToUpdate": "Failed to update ticket"
    },
    "settings": {
      "userIdMaintenance": {
        "title": "User ID Maintenance",
        "description": "Advance the users.id sequence or migrate one user ID across stored references.",
        "warning": "This is a high-risk maintenance action. Make sure the affected user is idle first. User ID changes update known references and caches, but external systems that stored the old ID will not be changed automatically.",
        "maxUserId": "Current Max ID",
        "nextUserId": "Next Auto ID",
        "sequenceName": "Sequence Name",
        "setNextTitle": "Set Current Auto-Increment Value",
        "setNextHint": "The value can only move forward. It must be greater than both the current next ID and the current maximum user ID.",
        "newNextUserId": "New Next User ID",
        "setNextValidation": "Enter a positive integer greater than the current next ID and current maximum user ID.",
        "setNextButton": "Update Next ID",
        "setNextSuccess": "User ID auto-increment value updated",
        "changeTitle": "Change a User ID Everywhere",
        "changeHint": "Migrates the users primary key and known FK/non-FK references. The current admin cannot change their own ID.",
        "oldUserId": "Old User ID",
        "newUserId": "New User ID",
        "confirmation": "Confirmation Text",
        "confirmationHint": "Type {text} to enable this action.",
        "changeValidation": "Enter valid, different old/new user IDs and the exact confirmation text.",
        "changeConfirm": "Change this user ID everywhere? References in multiple tables will be updated.",
        "changeButton": "Change User ID",
        "changeSuccess": "User ID changed",
        "submitting": "Processing..."
      },
      "site": {
        "supportTicketEntryVisibility": "Support Entry Visibility",
        "supportTicketEntryVisibilityHint": "When set to all users, the support entry appears in the user sidebar. When set to admin only, only the admin ticket management entry remains visible.",
        "supportTicketEntryVisibilityAll": "All Users",
        "supportTicketEntryVisibilityAdmin": "Admin Only",
        "siteSubtitleHint": "Displayed on login and register pages. Multiple lines are supported.",
        "siteDescription": "Site Description",
        "siteDescriptionPlaceholder": "A small service focused on stable access, not serving Mainland China users",
        "siteDescriptionHint": "Displayed in the homepage subtitle position. Multiple lines are supported. If empty, the homepage uses the site subtitle.",
        "apiKeyPageNotice": "API Keys Page Notice",
        "apiKeyPageNoticePlaceholder": "e.g., If it feels slow, use [code-us.urpg.net](https://code-us.urpg.net)\nDo not share your API key publicly.",
        "apiKeyPageNoticeHint": "Shown at the top of the API Keys page. Supports multi-line Markdown. Leave empty to hide it.",
        "customEndpoints": {
          "remove": "Remove",
          "moveUp": "Move Up",
          "moveDown": "Move Down"
        },
        "customHomeLinks": {
          "title": "Home Custom Links",
          "description": "Add custom links shown on the home page. Supports external http(s) URLs or internal paths starting with /.",
          "itemLabel": "Link #{n}",
          "linkTitle": "Title",
          "titlePlaceholder": "e.g., Documentation",
          "url": "URL",
          "urlPlaceholder": "https://docs.example.com or /dashboard",
          "descriptionLabel": "Description",
          "descriptionPlaceholder": "e.g., Learn how to use the API",
          "openInNewWindow": "Open in new window",
          "add": "Add Link",
          "remove": "Remove",
          "moveUp": "Move Up",
          "moveDown": "Move Down"
        }
      },
      "customMenu": {
        "description": "Add custom pages to the sidebar navigation and choose embedded display or external new-window opening.",
        "urlPlaceholder": "https://example.com/page?key={token}",
        "openMode": "Open Mode",
        "openModeEmbedded": "Embedded Page",
        "openModeExternal": "External Page",
        "openModeExternalConfirm": "External Page + Confirm Dialog",
        "group": "Menu Group",
        "groupPlaceholder": "Leave blank to use the default group"
      }
    }
  },
  "customPage": {
    "externalConfirmTitle": "Open external page",
    "externalConfirmMessage": "You are about to open the external page \"{label}\": {url}. Continue?",
    "externalConfirmOpen": "Continue",
    "externalOpenFailed": "Unable to open the external page. Check the menu URL configuration.",
    "imgKeyMissingTitle": "No image key available",
    "imgKeyMissingDesc": "Create or select an API key in an image-enabled OpenAI group first.",
    "imgKeyErrorTitle": "Failed to load image key",
    "imgKeyErrorDesc": "Unable to resolve {token}. Please try again later."
  }
} as const
