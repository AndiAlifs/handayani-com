import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

interface ChatMessage {
  text: string;
  sender: 'user' | 'bot';
  time: string;
}

@Component({
  selector: 'app-chat-bot',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './chat-bot.component.html',
  styleUrl: './chat-bot.component.css'
})
export class ChatBotComponent {
  isOpen = false;
  messageText = '';
  messages: ChatMessage[] = [
    {
      text: 'Halo! 👋 Saya asisten virtual YPA Handayani. Ada yang bisa saya bantu tentang program kursus kami?',
      sender: 'bot',
      time: this.getCurrentTime()
    }
  ];

  private botResponses = [
    'Terima kasih atas pertanyaannya! Untuk informasi lebih lanjut, silakan hubungi kami di WhatsApp 082191927620.',
    'Kami menawarkan kursus mengemudi, menjahit, komputer, dan bahasa. Anda tertarik program yang mana?',
    'Jadwal kursus mengemudi tersedia Senin - Sabtu, pukul 09.00 - 17.00. Ingin tahu detail lebih lanjut?',
    'Biaya kursus bervariasi tergantung program yang dipilih. Silakan lihat bagian "Program Kursus" di atas.',
    'Tim kami siap membantu Anda! Untuk pendaftaran langsung, klik tombol WhatsApp di halaman ini.'
  ];

  toggleChat(): void {
    this.isOpen = !this.isOpen;
  }

  sendMessage(): void {
    if (!this.messageText.trim()) return;

    this.messages.push({
      text: this.messageText,
      sender: 'user',
      time: this.getCurrentTime()
    });

    const userMsg = this.messageText;
    this.messageText = '';

    setTimeout(() => {
      const response = this.botResponses[Math.floor(Math.random() * this.botResponses.length)];
      this.messages.push({
        text: response,
        sender: 'bot',
        time: this.getCurrentTime()
      });
    }, 800 + Math.random() * 700);
  }

  private getCurrentTime(): string {
    return new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
  }
}
