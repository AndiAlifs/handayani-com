import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet],
  template: `<router-outlet></router-outlet>`,
  styles: [`:host { display: block; }`]
})
export class AppComponent {
  constructor(private translate: TranslateService) {
    this.translate.addLangs(['id', 'en']);
    this.translate.setDefaultLang('id');
    this.translate.use('id');
  }
}
