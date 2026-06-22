export type WhatsAppSessionStatus =
  | 'STOPPED'
  | 'STARTING'
  | 'SCAN_QR_CODE'
  | 'WORKING'
  | 'FAILED';

export interface WhatsAppStatus {
  sessionName: string;
  status: WhatsAppSessionStatus;
  phoneNumber: string;
  pairedAt: string | null;
  lastSyncedAt: string;
}

export interface WhatsAppMessage {
  id: number;
  direction: 'outbound' | 'inbound';
  phoneNumber: string;
  messageType: string;
  content: string;
  status: 'pending' | 'sent' | 'delivered' | 'read' | 'failed';
  context: string;
  createdAt: string;
}
