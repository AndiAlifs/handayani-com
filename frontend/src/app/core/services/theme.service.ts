import { Injectable, signal } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  private readonly THEME_KEY = 'user_theme';
  public isDarkMode = signal<boolean>(false);

  constructor() {
    this.initTheme();
  }

  private initTheme() {
    // Check local storage for preference
    const savedTheme = localStorage.getItem(this.THEME_KEY);
    
    if (savedTheme === 'dark') {
      this.isDarkMode.set(true);
      document.documentElement.classList.add('dark');
    } else {
      // By default it is light (also handles savedTheme === 'light')
      this.isDarkMode.set(false);
      document.documentElement.classList.remove('dark');
    }
  }

  public toggleTheme() {
    this.isDarkMode.update(dark => !dark);
    
    if (this.isDarkMode()) {
      document.documentElement.classList.add('dark');
      localStorage.setItem(this.THEME_KEY, 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem(this.THEME_KEY, 'light');
    }
  }
}
