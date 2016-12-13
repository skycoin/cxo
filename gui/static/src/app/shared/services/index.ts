import { SkyObjectService } from './skyobject.service';
import { BoardsService } from './boards.service';

// an array of services to resolve routes with data
export const APP_SERVICE_PROVIDERS = [
    SkyObjectService,
    BoardsService
];
