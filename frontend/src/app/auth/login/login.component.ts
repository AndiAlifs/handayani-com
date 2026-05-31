import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.css'
})
export class LoginComponent {
  username = '';
  password = '';
  isLoading = signal(false);
  errorMessage = signal('');
  showPassword = signal(false);

  constructor(
    private authService: AuthService,
    private router: Router
  ) {
    // If already logged in, redirect to dashboard
    if (this.authService.isAuthenticated()) {
      this.router.navigate(['/dashboard']);
    }
  }

  onLogin(): void {
    if (!this.username || !this.password) {
      this.errorMessage.set('Username dan password tidak boleh kosong.');
      return;
    }

    this.isLoading.set(true);
    this.errorMessage.set('');
    this.authService.login(this.username, this.password).subscribe((success) => {
      this.isLoading.set(false);
      if (success) {
        this.router.navigate(['/dashboard']);
      } else {
        this.errorMessage.set('Username atau password salah. Silakan coba lagi.');
      }
    });
  }

  togglePassword(): void {
    this.showPassword.update(v => !v);
  }
}
