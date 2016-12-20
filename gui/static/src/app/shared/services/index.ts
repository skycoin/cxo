import { SkyObjectService } from './skyobject.service';
import { BoardsService } from './boards.service';
import { SkyWireService } from './skywire.service';

// an array of services to resolve routes with data
export const APP_SERVICE_PROVIDERS = [
    SkyObjectService,
    SkyWireService,
    BoardsService
];
