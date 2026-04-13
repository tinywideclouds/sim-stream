import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActorMetersComponent } from './actor-meters.component';

describe('ActorMetersComponent', () => {
  let component: ActorMetersComponent;
  let fixture: ComponentFixture<ActorMetersComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ActorMetersComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ActorMetersComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});