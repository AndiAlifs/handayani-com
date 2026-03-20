import { Component } from '@angular/core';
import { HeroSectionComponent } from './hero-section/hero-section.component';
import { CoursePricingComponent } from './course-pricing/course-pricing.component';
import { InstructorScheduleComponent } from './instructor-schedule/instructor-schedule.component';
import { SimMechanismComponent } from './sim-mechanism/sim-mechanism.component';
import { ChatBotComponent } from './chat-bot/chat-bot.component';
import { CtaFooterComponent } from './cta-footer/cta-footer.component';

@Component({
  selector: 'app-landing-page',
  standalone: true,
  imports: [
    HeroSectionComponent,
    CoursePricingComponent,
    InstructorScheduleComponent,
    SimMechanismComponent,
    ChatBotComponent,
    CtaFooterComponent
  ],
  templateUrl: './landing-page.component.html',
  styleUrl: './landing-page.component.css'
})
export class LandingPageComponent {}
