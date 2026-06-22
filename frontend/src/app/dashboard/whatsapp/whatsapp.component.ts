import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
import { ApiService } from '../../core/services/api.service';
import { WhatsAppStatus, WhatsAppMessage } from '../../core/models/whatsapp.model';

@Component({
  selector: 'app-whatsapp',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './whatsapp.component.html',
  styleUrl: './whatsapp.component.css'
})
export class WhatsappComponent implements OnInit, OnDestroy {
  status: WhatsAppStatus | null = null;
  messages: WhatsAppMessage[] = [];

  qrUrl: SafeUrl | null = null;
  private qrObjectUrl: string | null = null;
  private qrTimer: ReturnType<typeof setInterval> | null = null;

  testPhone = '';
  testText = '';
  sending = false;
  sendNotice = '';
  busy = false;

  constructor(private api: ApiService, private sanitizer: DomSanitizer) {}

  ngOnInit() {
    this.loadStatus();
    this.loadMessages();
  }

  ngOnDestroy() {
    this.stopQrPolling();
    this.revokeQr();
  }

  loadStatus() {
    this.api.getWhatsAppStatus().subscribe(s => {
      this.status = s;
      if (s.status === 'SCAN_QR_CODE') {
        this.startQrPolling();
      } else {
        this.stopQrPolling();
        this.revokeQr();
      }
    });
  }

  loadMessages() {
    this.api.getMessageLog().subscribe(m => (this.messages = m));
  }

  start() { this.runAction(this.api.startWhatsApp()); }
  stop() { this.runAction(this.api.stopWhatsApp()); }
  restart() { this.runAction(this.api.restartWhatsApp()); }
  logout() { this.runAction(this.api.logoutWhatsApp()); }

  private runAction(obs: ReturnType<ApiService['startWhatsApp']>) {
    this.busy = true;
    obs.subscribe(s => {
      this.busy = false;
      this.status = s;
      if (s.status === 'SCAN_QR_CODE') this.startQrPolling();
      else { this.stopQrPolling(); this.revokeQr(); }
    });
  }

  private startQrPolling() {
    this.loadQr();
    if (this.qrTimer) return;
    this.qrTimer = setInterval(() => this.loadQr(), 5000);
  }

  private stopQrPolling() {
    if (this.qrTimer) {
      clearInterval(this.qrTimer);
      this.qrTimer = null;
    }
  }

  private loadQr() {
    this.api.getWhatsAppQR().subscribe({
      next: blob => {
        this.revokeQr();
        this.qrObjectUrl = URL.createObjectURL(blob);
        this.qrUrl = this.sanitizer.bypassSecurityTrustUrl(this.qrObjectUrl);
      },
      error: () => { /* WAHA down / not scannable — leave QR hidden */ }
    });
  }

  private revokeQr() {
    if (this.qrObjectUrl) {
      URL.revokeObjectURL(this.qrObjectUrl);
      this.qrObjectUrl = null;
    }
    this.qrUrl = null;
  }

  sendTest() {
    if (!this.testPhone || !this.testText) return;
    this.sending = true;
    this.sendNotice = '';
    this.api.sendTestMessage(this.testPhone, this.testText).subscribe(res => {
      this.sending = false;
      this.sendNotice = res ? 'Pesan terkirim.' : 'Gagal mengirim — periksa koneksi WAHA.';
      this.testText = '';
      this.loadMessages();
    });
  }

  badge(): { label: string; cls: string } {
    switch (this.status?.status) {
      case 'WORKING': return { label: '🟢 Terhubung', cls: 'ok' };
      case 'SCAN_QR_CODE': return { label: '🟡 Menunggu Scan QR', cls: 'warn' };
      case 'STARTING': return { label: '🟡 Memulai…', cls: 'warn' };
      default: return { label: '🔴 Terputus', cls: 'down' };
    }
  }
}
