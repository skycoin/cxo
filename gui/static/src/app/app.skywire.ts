//import {Component, OnInit, ViewChild} from 'app/angular2/core';
//import {ROUTER_DIRECTIVES, OnActivate} from 'app/angular2/router';
import {Component, OnInit, ViewChild} from 'angular2/core';
import {ROUTER_DIRECTIVES, OnActivate} from 'angular2/router';

import {Http, HTTP_BINDINGS, Response} from 'angular2/http';
import {HTTP_PROVIDERS, Headers} from 'angular2/http';
import {Observable} from 'rxjs/Observable';
import {Observer} from 'rxjs/Observer';
import 'rxjs/add/operator/map';
import 'rxjs/add/operator/catch';

declare var _: any;
declare var $: any;
declare var moment: any;

export class Node {
    pubKey: string;
    ip: string;
    port: string;
}

@Component({
    selector: 'load-skywire',
    directives: [ROUTER_DIRECTIVES],
    providers: [],
    templateUrl: 'app/templates/template.html'
})

export class loadComponent implements OnInit {
    //Declare default varialbes
    nodes: Array<any>;
    subscribers: Array<any>;
    subscriptions: Array<any>;

    newNodeSecKey: string;


    selectedNode: Node = {
        pubKey: "",
        ip: "",
        port: "",
    };

    constructor(private http: Http) { }

    //Init function for load default value
    ngOnInit() {
        this.nodes = [];
        this.subscribers = [];
        this.subscriptions = [];
        this.newNodeSecKey = "";
        this.loadNodeList();
    }

    loadNodeList() {
        $(".reloadNodeList").text("Loading...");

        var self = this;
        var headers = new Headers();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');
        var url = '/manager/nodes';
        this.http.get(url, { headers: headers })
            .map((res) => res.json())
            .subscribe(data => {
                console.log("get node list", url, data);
                $(".reloadNodeList").html('<i class="fa fa-refresh" aria-hidden="true">');
                if (data) {
                    self.nodes = data;
                } else {
                    return;
                }
            }, err => console.log("Error on load nodes: " + err), () => { });
    }
    addNewNode() {
        $("#modal-new-node").modal();
    }
    showNodeInfo(node: Node): void {
        this.selectedNode = node;
        this.loadSubscriberList(node);
        this.loadSubscriptionList(node);
        $("#modal-node-info").modal();
    }
    loadSubscriberList(node: Node): void {
        var self = this;
        var headers = new Headers();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');
        var url = '/manager/nodes/' + node.pubKey + '/subscribers';
        this.http.get(url, { headers: headers })
            .map((res) => res.json())
            .subscribe(data => {
                console.log("get subscriber list", url, data);
                if (data) {
                    self.subscribers = data;
                } else {
                    return;
                }
            }, err => console.log("Error on load subscribers: " + err), () => { });
    }
    loadSubscriptionList(node: Node): void {
        var self = this;
        var headers = new Headers();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');
        var url = '/manager/nodes/' + node.pubKey + '/subscriptions';
        this.http.get(url, { headers: headers })
            .map((res) => res.json())
            .subscribe(data => {
                console.log("get subscription list", url, data);
                if (data) {
                    self.subscriptions = data;
                } else {
                    return;
                }
            }, err => console.log("Error on load subscriptions: " + err), () => { });
    }

    createNewNode(): void {
        var self = this;
        var headers = new Headers();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');
        var url = '/manager/nodes/';

        var requestBody;
        if (this.newNodeSecKey.length === 0) {
          requestBody  = "";
        } else{
          requestBody  = JSON.stringify({ "secKey": this.newNodeSecKey });
        }

        console.log("sending to create new node");
        console.log(JSON.stringify(requestBody));
        this.http.post(url, requestBody, { headers: headers })
            .map((res) => res.json())
            .subscribe(data => {
                console.log("createNewNode", url, data);
                if (data.status == 200) {
                    console.log(data);
                    this.loadNodeList();
                    $("#modal-new-node").modal('hide');
                } else {
                    $("#new-node-error").removeClass("hide").text(data.detail);
                    return;
                }
            }, err => console.log("Error on createNewNode: " + err), () => { });
    }

    onKey(value: string) {
        this.newNodeSecKey = value;
    }
}
